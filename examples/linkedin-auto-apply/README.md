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
| 1 | validate | Check all required inputs |
| 2 | login | Chromium logs in to LinkedIn, session persisted |
| 3 | search-jobs | Navigate job search, extract listings via JS |
| 4 | analyze-jobs | LLM scores each job, filters by threshold, writes cover letter |
| 5 | apply-job | Per matched job: click Easy Apply, fill form, upload resume, submit |
| 6 | response | Return applied/skipped/failed summary |

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

## Run

```bash
kdeps run examples/linkedin-auto-apply
```

The API server starts at `http://localhost:16399`.

---

## API

### POST /api/v1/apply

**Required fields:**

| Field | Type | Notes |
|-------|------|-------|
| linkedin_email | string | LinkedIn login email |
| linkedin_password | string | LinkedIn login password |
| job_title | string | Search query |
| candidate_name | string | Used in cover letters and form fields |
| candidate_profile | string | Skills and experience summary for LLM scoring |
| resume_path | string | Absolute path to your resume PDF |

**Optional fields (used to answer Easy Apply form questions):**

| Field | Type | Default | Notes |
|-------|------|---------|-------|
| location | string | "" | Search location |
| min_match_score | number | 60 | Jobs below this are skipped |
| max_results | number | 5 | Max jobs to apply to |
| date_posted | string | r604800 | r86400=day, r604800=week, r2592000=month |
| remote_only | bool | false | Filter to remote jobs only |
| easy_apply_only | bool | true | Skip external-apply jobs |
| phone_number | string | "" | For phone form fields |
| years_of_experience | string | "" | For experience form fields |
| current_city | string | "" | For location form fields |
| country | string | Netherlands | For country select fields |
| require_visa | string | No | "Yes" or "No" for sponsorship questions |
| desired_salary | string | "" | For salary form fields |
| website | string | "" | Portfolio/GitHub URL |
| linkedin_url | string | "" | LinkedIn profile URL |

**Example request:**

```bash
curl -s -X POST http://localhost:16399/api/v1/apply \
  -H "Content-Type: application/json" \
  -d '{
    "linkedin_email":      "you@example.com",
    "linkedin_password":   "your_password",
    "job_title":           "Backend Engineer",
    "location":            "Amsterdam, Netherlands",
    "candidate_name":      "Jane Doe",
    "candidate_profile":   "8 years Go and Python. Kubernetes, AWS, Postgres. Led teams of 5.",
    "resume_path":         "/home/jane/resume.pdf",
    "min_match_score":     65,
    "phone_number":        "0612345678",
    "years_of_experience": "8",
    "current_city":        "Amsterdam",
    "country":             "Netherlands",
    "require_visa":        "No"
  }' | jq .
```

**Response:**

```json
[
  { "status": "applied",  "job_id": "3987654321", "title": "Senior Backend Engineer", "company": "Acme BV" },
  { "status": "skipped",  "job_id": "3987654322", "reason": "match score below threshold" },
  { "status": "failed",   "job_id": "3987654323", "reason": "no Easy Apply button" }
]
```

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

---

## Tuning

**Use a larger model** for better job scoring in `workflow.yaml`:
```yaml
model: deepseek-r1
```

**Lower the score threshold** to apply to more jobs:
```json
{ "min_match_score": 40 }
```

**Remote jobs only:**
```json
{ "remote_only": true }
```
