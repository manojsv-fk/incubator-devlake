# FourKites Engineering Signals – DevLake Fork

Branch: `manoj/fk-changes`

This branch extends the [Apache DevLake](https://devlake.apache.org/) open-source
engineering-metrics platform with FourKites-specific plugins for the **Engineering Signals
Hackathon 2026** (March 5–11).

The goal is to replace manual Jira-based reporting with an automated, AI-assisted metrics
platform covering the 22 P0 metrics across five categories.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          DevLake (this fork)                            │
│                                                                         │
│  ┌─────────────────── Existing plugins ──────────────────────┐         │
│  │  github  │  jenkins  │  jira  │  gitextractor  │  dora   │         │
│  └────────────────────────────────────────────────────────────┘         │
│                                                                         │
│  ┌──────────────── FourKites custom plugins (this branch) ────┐        │
│  │  aicode  (AI code impact metrics)                          │        │
│  │  chronosphere  (MTTR / Change Failure Rate)                │        │
│  └────────────────────────────────────────────────────────────┘        │
│                                                                         │
│  ┌─── Grafana dashboards (fourkites-config/grafana/) ─────────┐        │
│  │  Engineering Signals overview  │  DORA  │  AI Impact       │        │
│  └────────────────────────────────────────────────────────────┘        │
└─────────────────────────────────────────────────────────────────────────┘
```

**Data flow:** `Data Source → Collector → Extractor → Converter → Domain Layer → Grafana`

---

## P0 Metrics Coverage

| Category | Metric | Plugin | Status |
|---|---|---|---|
| Code Dev & AI | Commit Frequency | `github` | ✅ Built-in |
| Code Dev & AI | Code Churn | `github` | ✅ Built-in |
| Code Dev & AI | Code Complexity Score | `github` + gitextractor | ✅ Built-in |
| Code Dev & AI | **AI-Generated Lines of Code** | **`aicode`** | 🆕 Custom |
| PR & Review | Time Commit→PR | `github` | ✅ Built-in |
| PR & Review | Review Cycles | `github` | ✅ Built-in |
| PR & Review | Work Type Classification | `jira` | ✅ Built-in |
| Build, Test & Deploy | Deployment Frequency | `jenkins` + `dora` | ✅ Built-in |
| Build, Test & Deploy | Total Cycle Time | `jira` + `jenkins` + `dora` | ✅ Built-in |
| Build, Test & Deploy | Change Failure Rate | `jenkins` + **`chronosphere`** | 🆕 Custom |
| Post-Deploy | MTTR | **`chronosphere`** + `dora` | 🆕 Custom |
| Post-Deploy | Hotfix Deployments | `jenkins` | ✅ Built-in |
| Post-Deploy | Bugs Introduced | `jira` | ✅ Built-in |
| Post-Deploy | Bug Severity Tracking | `jira` | ✅ Built-in |
| Post-Deploy | Time on Features % | `jira` (worklogs) | ✅ Built-in |
| Post-Deploy | Time on Maintenance % | `jira` (worklogs) | ✅ Built-in |
| Business & DX | Unplanned Work % | `jira` | ✅ Built-in |
| Business & DX | Say/Do Ratio | `jira` | ✅ Built-in |
| Business & DX | Context Switching Freq | `jira` | ✅ Partial |

---

## Custom Plugins

### `aicode` — AI Code Impact Monitor
`backend/plugins/aicode/`

The most novel contribution of this branch. Tracks **AI-generated code** at FourKites by
combining two data sources:

**1. GitHub Copilot Metrics API** (`GET /orgs/{org}/copilot/metrics`)
- Daily suggestions / acceptances / lines accepted per language
- Computes `AcceptanceRate = acceptances / suggestions * 100%`
- Requires PAT with `manage_billing:copilot` or `read:org` scope

**2. Git Commit Trailer Analysis**
- Reads commits already collected by the `github`/`gitextractor` plugins
- Detects AI attribution trailers using configurable regex patterns:
  - `Co-authored-by: GitHub Copilot <copilot@github.com>` → `tool: "copilot"`
  - `Co-authored-by: Claude <noreply@anthropic.com>` → `tool: "claude"`
  - Custom patterns via `ScopeConfig.GenericAiTrailerPattern`
- Computes per-repo daily aggregates: `AiCodeRatio = AI lines / total lines`

**Tables produced:**
- `_tool_aicode_copilot_metrics` — daily org-level Copilot stats
- `_tool_aicode_commit_metrics` — per-commit AI attribution
- `_tool_aicode_daily_aggregates` — per-repo/day AI vs human lines

**Quickstart:**
```bash
# 1. Add connection via DevLake API or UI
POST /plugins/aicode/connections
{
  "name": "FourKites GitHub",
  "endpoint": "https://api.github.com/",
  "token": "<GitHub PAT>",
  "orgLogin": "fourkites"
}

# 2. Add scope
PUT /plugins/aicode/connections/1/scopes
[{ "login": "fourkites", "name": "FourKites" }]

# 3. Trigger collection
POST /pipelines
{
  "name": "aicode-test",
  "plan": [[{
    "plugin": "aicode",
    "options": { "connectionId": 1, "orgLogin": "fourkites" }
  }]]
}
```

---

### `chronosphere` — MTTR & Change Failure Rate
`backend/plugins/chronosphere/`

Connects to FourKites' [Chronosphere](https://chronosphere.io) observability platform to
collect alert events, enabling accurate **MTTR** and **Change Failure Rate** computation.

**What it collects:** `GET /api/v1/events` — alert firing and resolution events
**MTTR** = average time from `FiredAt` to `ResolvedAt` for production incidents
**Change Failure Rate** = % of deployments that triggered an alert within `changeFailureWindowMinutes`

The plugin links alert events to Jenkins deployment builds, allowing the built-in `dora`
plugin to compute CFR automatically.

**Tables produced:**
- `_tool_chronosphere_alert_events` — individual alert events with timestamps
- `_tool_chronosphere_incidents` — grouped incidents with MTTR

---

## Quick Setup

### Prerequisites
- Docker + Docker Compose
- GitHub PAT (scopes: `repo`, `read:org`, `manage_billing:copilot`)
- Jenkins API token (for a service account)
- Jira API token
- Chronosphere API token

### 1. Clone & switch to branch

```bash
git clone git@github.com:<your-fork>/incubator-devlake.git
cd incubator-devlake
git checkout manoj/fk-changes
```

### 2. Start DevLake

```bash
cp .env.example .env
# Edit .env – set ENCODE_KEY to a random 128-bit hex string
docker-compose up -d
```

DevLake UI: http://localhost:4000
Grafana: http://localhost:3002 (admin / admin)

### 3. Configure connections

Copy and fill in `fourkites-config/connections.example.yml` then use the DevLake UI
(Settings → Connections) to add each connection, or use the REST API directly.

See `fourkites-config/connections.example.yml` for all required fields.

### 4. Create a Blueprint

Import `fourkites-config/blueprint.example.json` into DevLake UI → Blueprints → Create.
Adjust `connectionId` values to match the connections you created in step 3.

Set the cron to `0 6 * * *` for a daily sync at 06:00 IST.

### 5. View dashboards

Open Grafana at http://localhost:3002. The following dashboards will show data after the
first successful pipeline run:

| Dashboard | Metrics |
|---|---|
| Engineering Signals Overview | All 22 P0 metrics |
| DORA Metrics | Deployment Frequency, CFR, MTTR, Lead Time |
| AI Code Impact | AI lines %, Copilot acceptance rate, per-tool breakdown |
| Code Quality | Commit frequency, code churn, PR cycle times |

---

## Development Notes

### Adding a new data source plugin

The DevLake plugin framework is well-documented in `backend/DevelopmentManual.md`.
The `aicode` and `chronosphere` plugins in this branch serve as FourKites-flavoured examples.

Every plugin follows: **Connection → Scope → ScopeConfig → Tasks (Collect → Extract → Convert)**

### Running a single plugin in standalone mode

```bash
cd backend/plugins/aicode
go run aicode.go --connectionId 1 --orgLogin fourkites
```

### Verifying plugin registration

```bash
curl http://localhost:8080/plugins | jq '.[] | .name' | grep -E 'aicode|chronosphere'
```

---

## Files Changed in This Branch

```
backend/plugins/aicode/           ← NEW: AI code impact metrics plugin
backend/plugins/chronosphere/     ← NEW: Chronosphere alert events plugin
fourkites-config/
  connections.example.yml         ← Example connection config for FourKites
  blueprint.example.json          ← Example DevLake blueprint
  grafana/
    FourKites-Engineering-Signals.json  ← NEW: All 22 P0 metrics overview dashboard
    FourKites-AI-Code-Impact.json       ← NEW: Copilot + Claude Code metrics dashboard
    README.md                           ← Import instructions
FOURKITES.md                      ← This file
```

---

## Team

FourKites Engineering Hackathon 2026 — Engineering Signals team
Hack Week: March 5–11 | Demo Day: March 12 | Awards: March 18
