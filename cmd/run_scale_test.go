// Copyright 2026 Kdeps, KvK 94834768
// Licensed under the Apache License, Version 2.0

package cmd

import (
	"context"
	"io"
	"net"
	stdhttp "net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v3/pkg/domain"
)

// TestScaleWorkerHelper is the fake worker the front tests start. It serves
// "hello" on KDEPS_BIND_PORT until SIGTERM, or exits at once with "fail".
func TestScaleWorkerHelper(_ *testing.T) {
	mode := os.Getenv("KDEPS_SCALE_HELPER")
	if mode == "" {
		return
	}
	if mode == "fail" {
		os.Exit(3)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:"+os.Getenv("KDEPS_BIND_PORT"))
	if err != nil {
		os.Exit(4)
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, os.Interrupt)
	go func() {
		<-sig
		os.Exit(0)
	}()
	_ = stdhttp.Serve(ln, stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		_, _ = io.WriteString(w, "hello")
	}))
	os.Exit(0)
}

func newTestFront(ctx context.Context, idle time.Duration) *scaleFront {
	return &scaleFront{
		ctx:          ctx,
		exe:          os.Args[0],
		args:         []string{"-test.run=^TestScaleWorkerHelper$"},
		idle:         idle,
		startTimeout: 20 * time.Second,
	}
}

func startTestFront(ctx context.Context, t *testing.T, f *scaleFront) (string, <-chan error) {
	t.Helper()
	port, err := freeLocalPort(ctx)
	require.NoError(t, err)
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	done := make(chan error, 1)
	go func() { done <- f.serve(ctx, addr) }()
	return addr, done
}

func get(t *testing.T, addr string) string {
	t.Helper()
	req, err := stdhttp.NewRequestWithContext(t.Context(), stdhttp.MethodGet, "http://"+addr+"/", nil)
	require.NoError(t, err)
	resp, err := stdhttp.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(body)
}

func (f *scaleFront) running() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.worker != nil
}

func TestScaleFront_WakeIdleWake(t *testing.T) {
	t.Setenv("KDEPS_SCALE_HELPER", "serve")
	ctx, cancel := context.WithCancel(t.Context())
	f := newTestFront(ctx, 300*time.Millisecond)
	addr, done := startTestFront(ctx, t, f)

	// The front starts the worker once at launch.
	require.Eventually(t, func() bool {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			_ = conn.Close()
		}
		return err == nil
	}, 20*time.Second, 20*time.Millisecond)
	assert.Equal(t, "hello", get(t, addr))

	require.Eventually(t, func() bool { return !f.running() }, 10*time.Second, 50*time.Millisecond,
		"idle stops the worker")
	assert.Equal(t, "hello", get(t, addr), "a request starts it again")
	assert.True(t, f.running())

	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(20 * time.Second):
		t.Fatal("the front did not stop")
	}
	assert.False(t, f.running())
}

func TestScaleFront_InFlightKeepsWorker(t *testing.T) {
	t.Setenv("KDEPS_SCALE_HELPER", "serve")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	f := newTestFront(ctx, time.Millisecond)
	_, err := f.acquire(ctx)
	require.NoError(t, err)
	f.reap(time.Now().Add(time.Hour))
	assert.True(t, f.running(), "a request in flight keeps the worker")
	f.release()
	f.reap(time.Now().Add(time.Hour))
	assert.False(t, f.running())
}

func TestScaleFront_StartFails(t *testing.T) {
	t.Setenv("KDEPS_SCALE_HELPER", "fail")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	f := newTestFront(ctx, time.Minute)
	_, done := startTestFront(ctx, t, f)
	select {
	case err := <-done:
		require.ErrorContains(t, err, "stopped before it opened its port")
	case <-time.After(30 * time.Second):
		t.Fatal("the front did not report the failed start")
	}
}

func TestScaleFront_ServeHTTPStartError(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	f := &scaleFront{ctx: ctx, exe: "/no/such/kdeps", startTimeout: time.Second}
	rec := &recorder{header: stdhttp.Header{}}
	f.ServeHTTP(rec, (&stdhttp.Request{}).WithContext(ctx))
	assert.Equal(t, stdhttp.StatusServiceUnavailable, rec.code)
	assert.False(t, f.running(), "the next request starts again")
	assert.Zero(t, f.active)
}

type recorder struct {
	header stdhttp.Header
	code   int
}

func (r *recorder) Header() stdhttp.Header      { return r.header }
func (r *recorder) Write(b []byte) (int, error) { return len(b), nil }
func (r *recorder) WriteHeader(code int)        { r.code = code }

func TestScaleFront_ListenError(t *testing.T) {
	f := &scaleFront{ctx: t.Context()}
	require.ErrorContains(t, f.serve(t.Context(), "256.0.0.1:1"), "server cannot start")
}

func TestWaitForPort(t *testing.T) {
	port, err := freeLocalPort(t.Context())
	require.NoError(t, err)
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	require.ErrorContains(t, waitForPort(t.Context(), addr, 100*time.Millisecond, nil), "did not open")
	exited := make(chan struct{})
	close(exited)
	require.ErrorContains(t, waitForPort(t.Context(), addr, time.Minute, exited), "stopped before")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, waitForPort(ctx, addr, time.Minute, nil), context.Canceled)
}

func TestStopProcessNil(t *testing.T) {
	require.NoError(t, stopProcess(nil))
}

func TestScaleFrontAddr(t *testing.T) {
	t.Setenv(scaleDirectEnv, "")
	api := &domain.APIServerConfig{HostIP: "127.0.0.1", PortNum: 18000 + os.Getpid()%1000}
	wf := func(s domain.WorkflowSettings) *domain.Workflow { return &domain.Workflow{Settings: s} }

	addr, err := scaleFrontAddr(wf(domain.WorkflowSettings{APIServer: api}), &RunFlags{})
	require.NoError(t, err)
	assert.Contains(t, addr, "127.0.0.1:")

	for name, tc := range map[string]struct {
		settings domain.WorkflowSettings
		flags    RunFlags
	}{
		"dev":       {domain.WorkflowSettings{APIServer: api}, RunFlags{DevMode: true}},
		"file":      {domain.WorkflowSettings{APIServer: api}, RunFlags{FileArg: "x"}},
		"fixed":     {domain.WorkflowSettings{APIServer: api, Scale: &domain.ScaleConfig{Min: 1, Max: 1}}, RunFlags{}},
		"no server": {domain.WorkflowSettings{}, RunFlags{}},
	} {
		t.Run(name, func(t *testing.T) {
			got, gotErr := scaleFrontAddr(wf(tc.settings), &tc.flags)
			require.NoError(t, gotErr)
			assert.Empty(t, got)
		})
	}

	_, err = scaleFrontAddr(wf(domain.WorkflowSettings{Scale: &domain.ScaleConfig{}}), &RunFlags{})
	require.ErrorContains(t, err, "needs apiServer or webServer")

	t.Setenv(scaleDirectEnv, "1")
	addr, err = scaleFrontAddr(wf(domain.WorkflowSettings{APIServer: api}), &RunFlags{})
	require.NoError(t, err)
	assert.Empty(t, addr)
}

func TestScaleIdle(t *testing.T) {
	api := &domain.APIServerConfig{PortNum: 8080}
	idle, err := scaleIdle(&domain.Workflow{Settings: domain.WorkflowSettings{
		APIServer: api, Scale: &domain.ScaleConfig{Idle: "3s"},
	}})
	require.NoError(t, err)
	assert.Equal(t, 3*time.Second, idle)

	idle, err = scaleIdle(&domain.Workflow{Settings: domain.WorkflowSettings{APIServer: api}})
	require.NoError(t, err)
	assert.Positive(t, idle, "the defaults asset sets the idle time")

	_, err = scaleIdle(&domain.Workflow{Settings: domain.WorkflowSettings{
		APIServer: api, Scale: &domain.ScaleConfig{Idle: "soon"},
	}})
	require.Error(t, err)
}

func TestWorkerArgs(t *testing.T) {
	c := &cobra.Command{Use: "run"}
	c.Flags().Bool("events", false, "")
	c.Flags().Bool("dev", false, "")
	c.Flags().String("file", "", "")
	c.Flags().StringSlice("tag", nil, "")
	c.Flags().Int("port", 1, "")
	require.NoError(t, c.Flags().Parse([]string{"--events", "--dev", "--file=x", "--tag=a,b"}))
	assert.Equal(t, []string{"run", "/w/workflow.yaml", "--events=true", "--tag=a", "--tag=b"},
		workerArgs(c, "/w/workflow.yaml"))
}

func TestRunScaleFront_BadIdle(t *testing.T) {
	c := &cobra.Command{}
	c.SetContext(t.Context())
	err := runScaleFront(c, &domain.Workflow{Settings: domain.WorkflowSettings{
		APIServer: &domain.APIServerConfig{}, Scale: &domain.ScaleConfig{Idle: "soon"},
	}}, "w.yaml", "127.0.0.1:0")
	require.ErrorContains(t, err, "deploy.scale.idle")
}

func TestExecuteWorkflowSteps_ServerStartsFront(t *testing.T) {
	t.Setenv(scaleDirectEnv, "")
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "resources"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "workflow.yaml"), []byte(`apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: front
  version: "1.0.0"
  targetActionId: a
deploy:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 18999
    routes:
      - path: /a
        methods: [GET]
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "resources", "a.yaml"),
		[]byte("actionId: a\nname: A\napiResponse:\n  success: true\n"), 0o644))
	orig := runScaleFrontFn
	t.Cleanup(func() { runScaleFrontFn = orig })
	var gotAddr string
	runScaleFrontFn = func(_ *cobra.Command, _ *domain.Workflow, _, addr string) error {
		gotAddr = addr
		return nil
	}
	c := &cobra.Command{}
	c.SetContext(t.Context())
	c.Flags().Bool("debug", false, "")
	require.NoError(t, ExecuteWorkflowStepsWithFlags(c, filepath.Join(tmp, "workflow.yaml"), &RunFlags{}))
	assert.Contains(t, gotAddr, "127.0.0.1:")
}
