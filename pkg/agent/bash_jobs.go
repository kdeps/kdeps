package agent

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const bashJobCmdMaxLen = 60 // max characters of command shown in job summaries

// bashJob tracks a bash command running in the background.
type bashJob struct {
	id      int
	command string
	started time.Time
	done    chan struct{}
	output  string
	err     error
}

// jobRegistry is the process-global registry of backgrounded bash jobs.
type jobRegistry struct {
	mu   sync.Mutex
	next int
	jobs map[int]*bashJob
}

//nolint:gochecknoglobals // process-wide registry; there is exactly one per agent process
var bashJobRegistry = &jobRegistry{
	jobs: make(map[int]*bashJob),
}

// add registers a background job and starts a goroutine to collect its output.
// waitCh must receive the result of cmd.Wait(). stdout/stderr are the live
// builders being written to by the still-running process.
func (r *jobRegistry) add(
	command string, stdout, stderr *strings.Builder, waitCh <-chan error,
) int {
	r.mu.Lock()
	r.next++
	id := r.next
	job := &bashJob{
		id:      id,
		command: command,
		started: time.Now(),
		done:    make(chan struct{}),
	}
	r.jobs[id] = job
	r.mu.Unlock()

	go func() {
		runErr := <-waitCh
		out := strings.TrimSpace(stdout.String())
		errOut := strings.TrimSpace(stderr.String())
		job.output = formatBashOutput(out, errOut)
		job.err = runErr
		close(job.done)
	}()
	return id
}

func (r *jobRegistry) get(id int) *bashJob {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.jobs[id]
}

func (r *jobRegistry) listAll() []*bashJob {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]*bashJob, 0, len(r.jobs))
	for _, j := range r.jobs {
		result = append(result, j)
	}
	return result
}

func (r *jobRegistry) remove(id int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.jobs, id)
}

// reset clears all jobs. Used in tests to isolate the global registry state.
func (r *jobRegistry) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs = make(map[int]*bashJob)
}

func (j *bashJob) status() string {
	select {
	case <-j.done:
		if j.err != nil {
			return "failed"
		}
		return "done"
	default:
		return "running"
	}
}

func (j *bashJob) summary() string {
	cmd := j.command
	if len(cmd) > bashJobCmdMaxLen {
		cmd = cmd[:bashJobCmdMaxLen-3] + "..."
	}
	elapsed := time.Since(j.started).Round(time.Second)
	return fmt.Sprintf("job %d [%s]: %s (started %s ago)", j.id, j.status(), cmd, elapsed)
}
