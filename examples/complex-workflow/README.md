# Complex Workflow Example

This example demonstrates a sophisticated multi-resource workflow with parallel execution, dependencies, and data flow between resources.

## Overview

This workflow analyzes GitHub repositories by:
1. Fetching repository data
2. Gathering commits, contributors, and issues (in parallel)
3. Analyzing activity with AI
4. Generating recommendations
5. Saving results to a file
6. Producing a final summary report

Perfect for demonstrating:
- **Complex dependencies** between resources
- **Parallel execution** of independent tasks
- **Data flow** through multiple steps
- **API orchestration** with multiple endpoints
- **AI analysis** of aggregated data
- **File output** and reporting

## Features Demonstrated

1. **8 Resources** - Complex multi-step workflow
2. **Parallel Execution** - Steps 2, 3, 4 run simultaneously
3. **Sequential Dependencies** - Each step waits for prerequisites
4. **HTTP Requests** - Multiple GitHub API calls
5. **LLM Analysis** - AI-powered insights
6. **File Operations** - Save results to disk
7. **Data Aggregation** - Combine data from multiple sources

## Prerequisites

### Install Ollama (for AI analysis)

```bash
# macOS/Linux
curl -fsSL https://ollama.com/install.sh | sh

# Pull the model
ollama pull llama3.2
```

## Running the Example

```bash
# Start the server
kdeps run examples/complex-workflow/workflow.yaml

# In another terminal, analyze a repository
curl -X POST http://localhost:16395/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "owner": "kubernetes",
    "repo": "kubernetes"
  }'
```

## Workflow Architecture

```
                [fetch-repo-data]
                        ↓
        ┌───────────────┼───────────────┐
        ↓               ↓               ↓
  [fetch-commits] [fetch-contributors] [fetch-issues]
        └───────────────┼───────────────┘
                        ↓
                [analyze-activity]
                        ↓
            [generate-recommendations]
                        ↓
                  [save-analysis]
                        ↓
                  [final-report]
```

### Execution Flow

1. **Step 1** (Sequential): Fetch basic repository data
2. **Steps 2-4** (Parallel): Fetch commits, contributors, and issues simultaneously
3. **Step 5** (Sequential): Analyze combined data with LLM
4. **Step 6** (Sequential): Generate recommendations
5. **Step 7** (Sequential): Save analysis to JSON file
6. **Step 8** (Sequential): Generate final summary

## Example Response

```json
{
  "success": true,
  "data": {
    "repository": "kubernetes/kubernetes",
    "analyzed_at": "2024-01-15T14:30:22Z",
    "stats": {
      "stars": 108234,
      "forks": 38912,
      "open_issues": 2567,
      "commits_analyzed": 10,
      "contributors": 5
    },
    "analysis": {
      "health": "Excellent - very active development",
      "engagement": "High community engagement with many contributors",
      "status": "Mature, production-ready project"
    },
    "recommendations": [
      {
        "title": "Issue Triage Automation",
        "description": "Implement automated labeling..."
      },
      {
        "title": "Contributor Onboarding",
        "description": "Create a comprehensive guide..."
      },
      {
        "title": "Documentation Updates",
        "description": "Regular documentation reviews..."
      }
    ]
  }
}
```

## Example Requests

### Analyze Kubernetes
```bash
curl -X POST http://localhost:16395/analyze \
  -H "Content-Type: application/json" \
  -d '{"owner": "kubernetes", "repo": "kubernetes"}'
```

### Analyze Docker
```bash
curl -X POST http://localhost:16395/analyze \
  -H "Content-Type: application/json" \
  -d '{"owner": "docker", "repo": "docker"}'
```

### Analyze Go Language
```bash
curl -X POST http://localhost:16395/analyze \
  -H "Content-Type: application/json" \
  -d '{"owner": "golang", "repo": "go"}'
```

## Understanding Dependencies

### Parallel Execution

These resources run **simultaneously** because they only depend on `fetch-repo-data`:

```yaml
- id: fetch-commits
  dependsOn: ["fetch-repo-data"]

- id: fetch-contributors
  dependsOn: ["fetch-repo-data"]

- id: fetch-issues
  dependsOn: ["fetch-repo-data"]
```

### Sequential Execution

This resource waits for **all three** previous steps to complete:

```yaml
- id: analyze-activity
  dependsOn: ["fetch-commits", "fetch-contributors", "fetch-issues"]
```

### Single Dependency

This resource only needs one prerequisite:

```yaml
- id: generate-recommendations
  dependsOn: ["analyze-activity"]
```

## Output Files

Analysis results are saved to `./output/` with filenames like:

```
./output/kubernetes_kubernetes_20240115_143022.json
```

Each file contains:
- Repository information
- Statistics (stars, forks, issues)
- AI analysis of activity
- Improvement recommendations

## Customization

### Add More Data Sources

Add additional parallel API calls:

```yaml
- id: fetch-releases
  dependsOn: ["fetch-repo-data"]
  config:
    url: "https://api.github.com/repos/{{input.owner}}/{{input.repo}}/releases"
```

### Modify Analysis Depth

Adjust the number of items fetched:

```yaml
params:
  per_page: "50"  # Fetch more commits
```

### Change AI Model

Use a different model for analysis:

```yaml
config:
  model: "mixtral"  # More powerful model
  temperature: 0.3  # More focused output
```

### Add Database Storage

Store results in a database instead of files:

```yaml
- id: save-to-db
  type: sql
  dependsOn: ["analyze-activity"]
  config:
    connection: "{{env.DATABASE_URL}}"
    query: "INSERT INTO analyses ..."
```

## Performance Notes

- **Parallel Execution**: Steps 2-4 run simultaneously, reducing total time by ~60%
- **Total Resources**: 8 resources (1 sequential, 3 parallel, 4 sequential)
- **Typical Runtime**: 10-15 seconds (depending on API latency)
- **Bottleneck**: LLM analysis (steps 5-6 take ~5 seconds each)

## Use Cases

### Repository Health Monitoring
Regularly analyze repositories to track health metrics over time.

### Open Source Project Evaluation
Evaluate projects before adoption by analyzing activity and community.

### DevOps Dashboards
Generate reports for management on project status.

### Automated Code Review
Trigger analysis on new releases or major changes.

## Troubleshooting

**GitHub API Rate Limits:**
- Unauthenticated: 60 requests/hour
- Authenticated: 5000 requests/hour

Add authentication:
```yaml
headers:
  Authorization: "token {{env.GITHUB_TOKEN}}"
```

**LLM Timeout:**
If analysis takes too long, reduce data size or use a faster model.

**Missing Output Directory:**
The workflow creates `./output/` automatically, but ensure write permissions.

## Next Steps

- Add GitHub authentication for higher rate limits
- Store results in a database for historical tracking
- Create visualizations of trends over time
- Add Slack/Discord notifications for reports
- Implement scheduling for regular analysis
- Combine with [webserver-static](../webserver-static/) for a dashboard
