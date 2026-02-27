# FourKites Grafana Dashboards

Import these dashboards into Grafana after your first successful DevLake pipeline run.

## Dashboards

| File | UID | Title | Description |
|------|-----|-------|-------------|
| `FourKites-Engineering-Signals.json` | `fk-eng-signals` | Engineering Signals Overview | All 22 P0 metrics in one view: Code Dev & AI, PR & Review, DORA, Post-Deploy, Business & DX |
| `FourKites-AI-Code-Impact.json` | `fk-ai-code-impact` | AI Code Impact | Copilot acceptance rate, AI-generated lines %, per-tool commit breakdown, user adoption trend |

## How to Import

**Via Grafana UI:**
1. Open Grafana at http://localhost:3002 (admin / admin)
2. Click the **+** icon → **Import**
3. Click **Upload JSON file** and select a file from this folder
4. Set the data source to **mysql** when prompted
5. Click **Import**

**Via Grafana API (scripted):**
```bash
for f in fourkites-config/grafana/*.json; do
  curl -s -X POST http://admin:admin@localhost:3002/api/dashboards/import \
    -H "Content-Type: application/json" \
    -d "{\"dashboard\": $(cat $f), \"overwrite\": true, \"folderId\": 0}"
  echo " ← imported $f"
done
```

## Variables

Both dashboards expose a **Days** template variable (7 / 14 / 30 / 90) to control the time window for all SQL queries.

The **AI Code Impact** dashboard also exposes a **Connection** variable that auto-populates from `_tool_aicode_connections`.

## Data Prerequisites

| Dashboard | Required plugins to run first |
|-----------|-------------------------------|
| Engineering Signals Overview | `github`, `jenkins`, `jira`, `dora`, `aicode`, `chronosphere` |
| AI Code Impact | `aicode` (+ `github`/`gitextractor` for commit trailer analysis) |
