# linkedin-auto-apply

AI-powered LinkedIn job search and application content generator.

Searches for relevant jobs, scores each listing against your profile
with a local LLM, and produces tailored cover letters for the best
matches - all running offline on your own machine.

---

## What it does

| Step | Resource | Description |
|------|----------|-------------|
| 1 | validate | Check required fields |
| 2 | search-jobs | Fetch listings from LinkedIn via JSearch API |
| 3 | parse-jobs | Normalise raw API response |
| 4 | analyze-jobs | LLM scores every job (0-100) against your profile |
| 5 | filter-jobs | Keep jobs above your minimum score threshold |
| 6 | generate-letters | LLM writes a tailored cover letter per matched job |
| 7 | assemble | Merge analysis + letters into final payload |
| 8 | response | Return structured JSON |

---

## Prerequisites

- kdeps installed (`kdeps --version`)
- [Ollama](https://ollama.ai) running locally
  ```
  ollama pull llama3.2
  ```
- A [RapidAPI](https://rapidapi.com) account with the
  [JSearch API](https://rapidapi.com/letscrape-6bRBa3QguO5/api/jsearch)
  subscribed (free tier: 200 requests/month)

---

## Run

```bash
# Export your RapidAPI key
export RAPIDAPI_KEY=your_key_here

# Start the agent
kdeps run examples/linkedin-auto-apply
```

The API server starts at `http://localhost:16399`.

---

## API

### POST /api/v1/apply

**Request body (JSON):**

```json
{
  "job_title":         "Software Engineer",
  "location":          "Amsterdam, Netherlands",
  "candidate_name":    "Jane Doe",
  "candidate_profile": "8 years Go and Python. Kubernetes, AWS, Postgres. Led teams of 5+. Open-source contributor.",
  "min_match_score":   65,
  "max_results":       10,
  "date_posted":       "week",
  "remote_only":       false
}
```

| Field | Type | Required | Default | Notes |
|-------|------|----------|---------|-------|
| job_title | string | yes | - | Search query sent to LinkedIn |
| location | string | no | - | City, country, or region |
| candidate_name | string | yes | - | Used in cover letters |
| candidate_profile | string | yes | - | Skills and experience summary |
| min_match_score | number | no | 60 | 0-100; jobs below this are dropped |
| max_results | number | no | 10 | Max listings to fetch from API |
| date_posted | string | no | week | today, 3days, week, month |
| remote_only | bool | no | false | Filter to remote-only jobs |
| rapidapi_key | string | no | $RAPIDAPI_KEY | Override env var per request |

**Response (JSON):**

```json
{
  "matched_jobs": [
    {
      "title":         "Senior Software Engineer",
      "company":       "Acme BV",
      "location":      "Amsterdam, Netherlands",
      "apply_link":    "https://www.linkedin.com/jobs/view/...",
      "is_remote":     false,
      "match_score":   87,
      "match_reasons": ["Go expertise matches core requirement", "Kubernetes experience aligns with platform role"],
      "gaps":          ["Rust experience preferred but not required"],
      "cover_letter":  "Dear Acme BV Hiring Team,\n\nYour Senior Software Engineer..."
    }
  ],
  "total_searched": 10,
  "total_matched":  3
}
```

---

## Example curl

```bash
curl -s -X POST http://localhost:16399/api/v1/apply \
  -H "Content-Type: application/json" \
  -d '{
    "job_title": "Backend Engineer",
    "location": "Amsterdam, Netherlands",
    "candidate_name": "Jane Doe",
    "candidate_profile": "8 years Go and Python. Kubernetes, AWS, Postgres. Led teams of 5.",
    "min_match_score": 65
  }' | jq .
```

---

## Differences from Selenium-based auto-apply bots

| Feature | Selenium bot | This workflow |
|---------|-------------|---------------|
| Job search | Browser automation | JSearch API |
| Form filling | Selenium click-through | Not included - cover letters are ready to paste |
| Resume tailoring | AI rewrite | Match reasons highlight what to emphasise |
| Cover letter | AI generated | Per-job, personalised |
| Infrastructure | Needs Chrome + driver | Runs headless, no browser |
| Rate limits | Account ban risk | API rate limited, no ToS violation |
| Runs offline | No (needs LinkedIn session) | LLM analysis is fully offline |

This workflow focuses on the research and content-generation steps.
Once you have the `cover_letter` and `apply_link` for each matched job,
manual submission (or a separate automation layer) applies in seconds.

---

## Tuning

**Increase match accuracy** - use a larger model:
```yaml
models:
  - llama3.2:latest   # or deepseek-r1, mistral, etc.
```

**Lower the score threshold** to see more options:
```json
{ "min_match_score": 40 }
```

**Restrict to remote jobs** only:
```json
{ "remote_only": true }
```
