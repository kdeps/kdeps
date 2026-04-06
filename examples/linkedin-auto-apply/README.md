# linkedin-auto-apply

Automated LinkedIn job applicant powered by a headless Chromium browser and a local LLM.

Replicates the full `auto-apply/` bot natively using kdeps browser resources:
logs in, searches jobs with filters, scores each listing against your profile,
and runs the complete Easy Apply form flow (fill questions, upload resume,
submit) for every matched job.

---

## Pipeline

| Step | Resource | What it does |
|------|----------|--------------|
| 1 | load-profile | Parse YAML front matter from the file input source |
| 2 | set-profile | Populate kdeps memory from parsed profile JSON |
| 3 | validate | Check all required inputs |
| 4 | login | Chromium logs in to LinkedIn, session persisted |
| 5 | search-jobs | Navigate job search, extract listings via JS |
| 6 | analyze-jobs | LLM scores each job, filters by threshold, writes cover letter |
| 7 | apply-job | Per matched job: click Easy Apply, fill form, upload resume, submit |
| 8 | response | Return applied/skipped/failed summary |

---

## Prerequisites

- kdeps installed (`kdeps --version`)
- [Ollama](https://ollama.ai) running locally with `llama3.2`
  ```
  ollama pull llama3.2
  ```
- A LinkedIn account with Easy Apply access
- Your resume as a local PDF file

---

## Profile file

Copy `sample-profile.md` to `joel-profile.md` (or any name) and fill in your details.
The file uses YAML front matter between `---` delimiters:

```markdown
---
job_title: "Senior Software Engineer"
location: "Amsterdam, Netherlands"
candidate_name: "Jane Doe"
linkedin_email: "jane@example.com"
linkedin_password: "your-password"
resume_path: "/Users/jane/resume.pdf"
# ... see sample-profile.md for all fields
---
```

---

## Run

Pass the profile file via `--file`, stdin, or `KDEPS_FILE_PATH`:

```bash
# --file flag (recommended)
kdeps run examples/linkedin-auto-apply --file joel-profile.md

# stdin pipe
cat joel-profile.md | kdeps run examples/linkedin-auto-apply

# environment variable
KDEPS_FILE_PATH=joel-profile.md kdeps run examples/linkedin-auto-apply
```

The workflow reads the markdown file, parses the YAML front matter, and runs the
full browser automation pipeline without any API server or HTTP POST required.

---

## How it compares to the Selenium bot

| Feature | Selenium bot (`auto-apply/`) | This workflow |
|---------|------------------------------|---------------|
| Login | Selenium + credentials | Playwright/Chromium + credentials |
| Job search | `driver.get(search URL)` | Browser resource navigate |
| Filters | Click UI filter elements | URL params (f_TPR, f_WT, f_LF) |
| Job scoring | Not built-in | LLM scores 0-100 against profile |
| Cover letter | Static text from config | LLM generates per-job |
| Easy Apply | Selenium click-through | Browser evaluate JS |
| Form filling | Label-match + config values | Label-match + config values (same logic) |
| Resume upload | `input.send_keys(path)` | Browser upload action |
| Submit | `wait_span_click("Submit")` | JS click submit button |
| Already-applied check | Job state footer text | JS footer text check |
| Session | Selenium WebDriver | Playwright sessionId |
| Profile input | Python config files | File input source (--file / stdin) |

---

## Tuning

**Use a larger model** for better job scoring in `workflow.yaml`:
```yaml
model: deepseek-r1
```

**Lower the score threshold** in your profile file:
```yaml
min_match_score: "40"
```

**Remote jobs only:**
```yaml
remote_only: "true"
```

