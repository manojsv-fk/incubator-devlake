# Engineering Signals — P0 Metrics Implementation Plan

> **Scope:** 16 metrics (7 "done" + 9 "partial")
> **Hackathon deadline:** Demo Day March 12, 2026

---

## Ground Truth: What's Actually Working vs. Broken

After reading the actual domain layer models (`pull_request.go`, `issue.go`, `sprint.go`,
`cicd_deployment.go`, `project_pr_metric.go`), the real state is:

| # | Metric | Claimed | Reality | Root Cause |
|---|---|---|---|---|
| 1 | Commit Frequency | ✅ Done | ✅ **Working** | SQL is correct |
| 2 | AI-Generated Lines % | ✅ Done | ✅ **Working** | `aicode` plugin correct |
| 3 | MTTR | ✅ Done | ✅ **Working** | `chronosphere` plugin correct |
| 4 | Bug Severity Distribution | ✅ Done | ✅ **Working** | `issues.urgency` field exists |
| 5 | Deployment Frequency | ✅ Done | ✅ **Working** | `cicd_deployments` query is correct |
| 6 | Context Switching Freq | ⚠️ Partial | ✅ **Working** | `issues.epic_key`, `assignee_id`, `YEARWEEK` all exist |
| 7 | Time Commit→PR | ✅ Done | ❌ **Broken SQL** | `pull_requests.first_committed_at` doesn't exist — data is in `project_pr_metrics.first_commit_authored_date` |
| 8 | Total Cycle Time | ✅ Done | ❌ **Broken SQL** | `pull_requests.lead_time_minutes` doesn't exist — data is in `project_pr_metrics.pr_cycle_time` |
| 9 | Review Cycles | ⚠️ Partial | ❌ **Broken SQL** | `pull_requests.review_rounds` was removed from GitLab, never existed in GitHub |
| 10 | Work Type Classification | ⚠️ Partial | ❌ **Broken SQL** | `issues.type_dev_cost` doesn't exist — field is `issues.type` |
| 11 | Unplanned Work % | ⚠️ Partial | ❌ **Broken SQL** | `issues.is_bug` doesn't exist — should be `issues.type = 'BUG'` |
| 12 | Say/Do Ratio | ⚠️ Partial | ❌ **Broken SQL** | `issues.sprint_id` doesn't exist — sprint relationship is in `sprint_issues` join table |
| 13 | Change Failure Rate | ⚠️ Partial | ❌ **Broken SQL** | `cicd_deployments.change_failure` column doesn't exist in the domain model |
| 14 | Code Churn | ⚠️ Partial | ⚠️ **Wrong formula** | SQL runs but measures deletion ratio, not the 14-day rework signal in the spec |
| 15 | Hotfix Deployments | ⚠️ Partial | ⚠️ **Fragile** | SQL runs but relies entirely on Jenkins job naming |
| 16 | Bugs Introduced | ⚠️ Partial | ⚠️ **Counts only** | `issues.type` is correct but no PR/commit linkage |

**Net result: 6 fully working, 7 broken SQL panels (return NULL or error), 3 weak proxies.**

---

## Implementation Order

```
Day 1  (Mar 5)  SQL fixes only — no Go code                     ~3 hrs
Day 2  (Mar 6)  DevLake configuration + pipeline re-run         ~3 hrs
Day 3  (Mar 7)  CFR Go subtask + registration                   ~1 day
Day 4  (Mar 8)  Strengthen weak panels + linkage                ~3 hrs
Day 5  (Mar 11) Polish + demo rehearsal
Mar 12          DEMO DAY
```

---

## Group 1: SQL-Only Fixes

All changes are to `fourkites-config/grafana/FourKites-Engineering-Signals.json`.
No Go code required.

---

### Fix 1 — Time Commit→PR

**Panel:** `"Time Commit → PR (avg min)"`

**Problem:** `pull_requests.first_committed_at` does not exist.
The DORA `calculateChangeLeadTime` task computes this and writes it to
`project_pr_metrics.first_commit_authored_date`.

**Replace `rawSql` with:**
```sql
SELECT
  ROUND(
    AVG(TIMESTAMPDIFF(MINUTE, m.first_commit_authored_date, m.pr_created_date))
  , 0) AS time_commit_to_pr_min
FROM project_pr_metrics m
WHERE m.pr_created_date >= DATE_SUB(NOW(), INTERVAL $__interval_days DAY)
  AND m.first_commit_authored_date IS NOT NULL
  AND m.pr_created_date > m.first_commit_authored_date
```

**Prerequisite:** A DevLake **Project** must exist with GitHub repo + Jenkins pipeline
scopes mapped. `project_pr_metrics` is only populated when DORA's
`calculateChangeLeadTime` subtask runs against a named project.

---

### Fix 2 — Total Cycle Time

**Panel:** `"Total Cycle Time (avg, min)"`

**Problem:** `pull_requests.lead_time_minutes` does not exist.
DORA stores cycle time in `project_pr_metrics.pr_cycle_time` (coding + review + deploy,
in minutes).

**Replace `rawSql` with:**
```sql
SELECT
  ROUND(AVG(m.pr_cycle_time), 0) AS total_cycle_time_min
FROM project_pr_metrics m
WHERE m.pr_merged_date >= DATE_SUB(NOW(), INTERVAL $__interval_days DAY)
  AND m.pr_cycle_time IS NOT NULL
  AND m.pr_cycle_time > 0
```

**Optional — add a breakdown stat panel (high demo value):**

| Sub-metric | Column | SQL snippet |
|---|---|---|
| Coding time | `pr_coding_time` | `SELECT ROUND(AVG(pr_coding_time),0) FROM project_pr_metrics WHERE ...` |
| Pickup time (PR open → first review) | `pr_pickup_time` | same pattern |
| Review time (first review → merge) | `pr_review_time` | same pattern |
| Deploy time (merge → deploy) | `pr_deploy_time` | same pattern |

---

### Fix 3 — Review Cycles

**Panel:** `"Review Cycles (avg per PR)"`

**Problem:** `pull_requests.review_rounds` was removed from the GitLab plugin in
migration `20240904_remove_mr_review_fields.go` and was never added to the GitHub
plugin. The column does not exist.

**Compute from `pull_request_comments` instead (most accurate):**
```sql
SELECT
  ROUND(AVG(review_count), 1) AS review_cycles
FROM (
  SELECT
    prc.pull_request_id,
    COUNT(DISTINCT DATE(prc.created_date)) AS review_count
  FROM pull_request_comments prc
  JOIN pull_requests pr ON pr.id = prc.pull_request_id
  WHERE pr.merged_date >= DATE_SUB(NOW(), INTERVAL $__interval_days DAY)
    AND pr.status     = 'MERGED'
    AND prc.account_id != pr.author_id  -- exclude author's own comments
  GROUP BY prc.pull_request_id
  HAVING review_count > 0
) t
```

**Prerequisite:** The GitHub scope config must have "Collect Pull Request Comments"
enabled (subtask `collectPrComments`). Go to DevLake UI → GitHub connection →
scope → Subtasks and confirm it is checked.

---

### Fix 4 — Work Type Classification

**Panel:** `"Work Type Classification"`

**Problem:** `issues.type_dev_cost` does not exist. The domain `Issue` struct uses
`Type` with constants `BUG`, `REQUIREMENT`, `INCIDENT`, `TASK`, `SUBTASK`.

**Replace `rawSql` with:**
```sql
SELECT
  CASE
    WHEN type = 'REQUIREMENT' THEN 'Feature'
    WHEN type = 'BUG'         THEN 'Bug Fix'
    WHEN type = 'INCIDENT'    THEN 'Incident'
    WHEN type = 'TASK'        THEN 'Task'
    ELSE COALESCE(original_type, 'Other')
  END AS work_type,
  COUNT(*) AS issues
FROM issues
WHERE created_date >= DATE_SUB(NOW(), INTERVAL $__interval_days DAY)
GROUP BY work_type
ORDER BY issues DESC
```

**Prerequisite:** Jira scope config issue type mappings must be configured in DevLake
UI. Recommended mapping:

| Jira Type | DevLake Type |
|---|---|
| Story | REQUIREMENT |
| Bug | BUG |
| Epic | REQUIREMENT |
| Task | TASK |
| Sub-task | SUBTASK |
| Incident | INCIDENT |

---

### Fix 5 — Unplanned Work %

**Panel:** `"Unplanned Work %"`

**Problem:** `issues.is_bug` does not exist in the domain `Issue` model. Use
`issues.type`.

**Replace `rawSql` with:**
```sql
SELECT
  ROUND(
    SUM(CASE WHEN type IN ('BUG', 'INCIDENT') THEN 1 ELSE 0 END)
    / NULLIF(COUNT(*), 0) * 100
  , 1) AS unplanned_pct
FROM issues
WHERE created_date >= DATE_SUB(NOW(), INTERVAL $__interval_days DAY)
```

> **Note:** This remains a proxy (all bugs ≠ unplanned). The sprint-aware version
> requires Fix 6 to be complete first. Once sprint data is available, replace with:
> ```sql
> -- Sprint-aware unplanned work: issues added AFTER sprint start date
> SELECT
>   ROUND(
>     SUM(CASE WHEN i.created_date > s.started_date THEN 1 ELSE 0 END)
>     / NULLIF(COUNT(*), 0) * 100
>   , 1) AS unplanned_pct
> FROM issues i
> JOIN sprint_issues si ON si.issue_id = i.id
> JOIN sprints s        ON s.id = si.sprint_id
> WHERE s.completed_date >= DATE_SUB(NOW(), INTERVAL $__interval_days DAY)
>   AND s.status = 'CLOSED'
> ```

---

### Fix 6 — Say/Do Ratio

**Panel:** `"Say/Do Ratio %"`

**Problem:** `issues.sprint_id` does not exist in the domain `Issue` model. The
sprint–issue relationship lives in the `sprint_issues` join table
(`SprintId`, `IssueId`). Also `sprints.completed_date` is the correct column name.

**Replace `rawSql` with:**
```sql
SELECT
  ROUND(
    SUM(CASE WHEN i.status = 'DONE' THEN 1 ELSE 0 END)
    / NULLIF(COUNT(*), 0) * 100
  , 1) AS say_do_ratio
FROM issues i
JOIN sprint_issues si ON si.issue_id = i.id
JOIN sprints s        ON s.id = si.sprint_id
WHERE s.completed_date >= DATE_SUB(NOW(), INTERVAL $__interval_days DAY)
  AND s.status = 'CLOSED'
```

**Prerequisite:** Jira sprint collection must be enabled. In DevLake UI, Jira scope
config → enable subtasks `collectSprints`, `extractSprints`, `convertSprints`,
`convertSprintIssues`. Verify: `SELECT COUNT(*) FROM sprints` must return > 0 after
a pipeline run.

---

## Group 2: Go Code + SQL (Change Failure Rate)

### Fix 7 — Change Failure Rate

**Two parts:** a new Go subtask + a dashboard SQL fix.

#### Part A — New file: `backend/plugins/chronosphere/tasks/convert_alerts_to_cfr.go`

**Problem:** `cicd_deployments.change_failure` does not exist in the `CICDDeployment`
struct. DevLake's DORA plugin computes CFR via the `incidents` domain table +
`project_incident_deployment_relationships`. The `chronosphere` plugin collects alert
events but never promotes them to `incidents`, so the DORA chain is broken.

**What the new subtask must do:**

1. Read `_tool_chronosphere_alert_events` WHERE `status = 'resolved'`
   (or firing events with a non-nil `resolved_at`)
2. For each event, write a row to the domain `ticket.Incident` table:
   - `CreatedDate = event.FiredAt`
   - `ResolutionDate = event.ResolvedAt`
   - `Status = 'DONE'`
   - `Type = 'INCIDENT'`
   - `Title = event.AlertName`
   - `OriginalStatus = event.Status`
   - `ScopeId` = Chronosphere connection namespace (so project mapping works)
3. After this subtask runs, the existing DORA `ConnectIncidentToDeployment` task
   automatically joins each incident to the most recent successful deployment that
   finished before `incident.CreatedDate`, writing to
   `project_incident_deployment_relationships`.

**Register in `backend/plugins/chronosphere/impl/impl.go`:**
- Add `tasks.ConvertAlertsToCfrMeta` to the plugin's `SubTaskMetas()` list after
  `ExtractAlertEventsMeta`.

**Update `backend/plugins/chronosphere/tasks/task_data.go`:**
- No new fields needed; the subtask reads existing `_tool_chronosphere_alert_events`
  and writes to the domain `incidents` table.

#### Part B — Fix dashboard SQL

**Replace `rawSql` in panel `"Change Failure Rate %"` with:**
```sql
SELECT
  ROUND(
    COUNT(DISTINCT r.deployment_id)
    / NULLIF(
        (SELECT COUNT(*)
         FROM cicd_deployments
         WHERE created_date >= DATE_SUB(NOW(), INTERVAL $__interval_days DAY)
           AND result = 'SUCCESS'),
        0
      ) * 100
  , 1) AS cfr_pct
FROM project_incident_deployment_relationships r
JOIN incidents i ON i.id = r.id
WHERE i.created_date >= DATE_SUB(NOW(), INTERVAL $__interval_days DAY)
```

**Prerequisite:** DORA plugin must run after `chronosphere`. Update the Blueprint to
include a DORA stage after the chronosphere stage:
```json
[
  [{ "plugin": "chronosphere", "options": { ... } }],
  [{ "plugin": "dora",         "options": { "projectName": "fourkites" } }]
]
```

---

## Group 3: SQL Formula Improvements

### Improve: Code Churn

**Current formula:** `SUM(additions + deletions) / total` — measures deletion ratio.

**Spec definition:** Lines deleted *within 14 days* of being added to the same file
(rework/instability signal).

**Replace `rawSql` with:**
```sql
SELECT
  ROUND(
    SUM(cf_del.deletions)
    / NULLIF(SUM(cf_add.additions), 0) * 100
  , 1) AS churn_pct
FROM commit_files cf_add
JOIN commits c_add  ON c_add.sha  = cf_add.commit_sha
JOIN commit_files cf_del ON cf_del.file_path = cf_add.file_path
JOIN commits c_del  ON c_del.sha  = cf_del.commit_sha
WHERE c_add.authored_date >= DATE_SUB(NOW(), INTERVAL $__interval_days DAY)
  AND c_del.authored_date BETWEEN c_add.authored_date
                               AND DATE_ADD(c_add.authored_date, INTERVAL 14 DAY)
  AND c_del.sha    != c_add.sha
  AND cf_del.deletions > 0
```

> **Performance note:** This self-join is expensive on large repos. For the demo,
> constrain to a single `repo_id`. For production, materialize results nightly into
> a `_fk_code_churn_daily` table (same pattern as `_tool_aicode_daily_aggregates`).

**Add a trend panel alongside it (high demo value):**
```sql
SELECT
  DATE(c.authored_date)  AS time,
  SUM(cf.deletions)      AS lines_deleted,
  SUM(cf.additions)      AS lines_added
FROM commits c
JOIN commit_files cf ON cf.commit_sha = c.sha
WHERE c.authored_date >= DATE_SUB(NOW(), INTERVAL $__interval_days DAY)
GROUP BY DATE(c.authored_date)
ORDER BY time ASC
```

---

### Improve: Hotfix Deployments

**Current:** `WHERE name REGEXP '(?i)hotfix'` — single keyword, misses rollbacks,
reverts, and emergency patches.

**Replace `rawSql` with:**
```sql
SELECT
  COUNT(*) AS hotfix_deploys
FROM cicd_deployments
WHERE created_date >= DATE_SUB(NOW(), INTERVAL $__interval_days DAY)
  AND (
       name          REGEXP '(?i)(hotfix|hot.?fix|rollback|revert|emergency|patch)'
    OR display_title REGEXP '(?i)(hotfix|hot.?fix|rollback|revert|emergency|patch)'
  )
```

**Long-term path:** In Jenkins pipelines, pass `devlake-deployment-env=HOTFIX` as a
build parameter so DevLake's Jenkins plugin sets `environment = 'HOTFIX'` on the
`cicd_deployments` row. Then the query becomes a simple
`WHERE environment = 'HOTFIX'` — no regex fragility.

---

### Improve: Bugs Introduced (add PR linkage panel)

**Current panel:** Counts BUG-type issues. Correct but shows no causation.

**Keep existing count panel.** Add a second table panel alongside it:

```sql
-- Bugs traceable to the PR that shipped them
-- Requires: linker plugin stage in the blueprint
SELECT
  pr.title        AS pull_request,
  pr.url          AS pr_url,
  i.title         AS bug_title,
  i.urgency       AS severity,
  i.created_date  AS bug_found_date,
  pr.merged_date  AS shipped_date
FROM issues i
JOIN pull_request_issues pri ON pri.issue_id = i.id
JOIN pull_requests pr        ON pr.id = pri.pull_request_id
WHERE i.type = 'BUG'
  AND i.created_date >= DATE_SUB(NOW(), INTERVAL $__interval_days DAY)
ORDER BY i.urgency DESC, i.created_date DESC
LIMIT 20
```

**Prerequisite:** The `linker` plugin must run after both GitHub and Jira. It matches
Jira issue keys mentioned in commit messages or PR descriptions. Add to blueprint:
```json
[
  [{ "plugin": "github", "options": { ... } }],
  [{ "plugin": "jira",   "options": { ... } }],
  [{ "plugin": "linker", "options": { "projectName": "fourkites" } }]
]
```

---

## Group 4: Configuration Prerequisites

These metrics have correct SQL (after the fixes above) but require specific DevLake
setup that must happen before a pipeline run.

### 4.1 Create a DevLake Project

Required by: **Time Commit→PR**, **Total Cycle Time**, **Change Failure Rate**

`project_pr_metrics` and `project_incident_deployment_relationships` are only
populated when DORA runs against a named project.

Steps:
1. DevLake UI → Projects → Create Project → name it `fourkites`
2. Add scopes: GitHub repo(s) + Jenkins pipeline(s)
3. Run the DORA plugin for the project

### 4.2 Enable PR Comment Collection (GitHub)

Required by: **Review Cycles**

DevLake UI → Connections → GitHub → `<your connection>` → Scopes →
`<your repo>` → Subtask settings → enable **"Collect Pull Request Comments"**
(`collectPrComments`).

### 4.3 Configure Jira Issue Type Mappings

Required by: **Work Type Classification**, **Unplanned Work %**

DevLake UI → Connections → Jira → `<your connection>` → Scope Config →
Issue Type Mapping:

| Jira Type | DevLake Standard Type |
|---|---|
| Story | REQUIREMENT |
| Bug | BUG |
| Epic | REQUIREMENT |
| Task | TASK |
| Sub-task | SUBTASK |
| Incident / Service Request | INCIDENT |

### 4.4 Enable Jira Sprint Collection

Required by: **Say/Do Ratio**, **Unplanned Work % (sprint-aware)**

DevLake UI → Connections → Jira → `<your connection>` → Subtask settings →
enable: `collectSprints`, `extractSprints`, `convertSprints`, `convertSprintIssues`.

Verify after run: `SELECT COUNT(*) FROM sprints` must return > 0.

### 4.5 Add Linker Plugin to Blueprint

Required by: **Bugs Introduced (PR linkage panel)**

Add a `linker` stage to `fourkites-config/blueprint.example.json` after the
`github` and `jira` stages.

---

## End State After Plan Execution

| # | Metric | Before | After Plan |
|---|---|---|---|
| 1 | Commit Frequency | ✅ Working | ✅ No change needed |
| 2 | AI-Generated Lines % | ✅ Working | ✅ No change needed |
| 3 | MTTR | ✅ Working | ✅ No change needed |
| 4 | Bug Severity Distribution | ✅ Working | ✅ No change needed |
| 5 | Deployment Frequency | ✅ Working | ✅ No change needed |
| 6 | Context Switching Freq | ✅ Working | ✅ No change needed |
| 7 | Time Commit→PR | ❌ NULL | ✅ SQL → `project_pr_metrics` |
| 8 | Total Cycle Time | ❌ NULL | ✅ SQL → `project_pr_metrics` + breakdown |
| 9 | Review Cycles | ❌ NULL | ✅ SQL → `pull_request_comments` |
| 10 | Work Type Classification | ❌ NULL | ✅ SQL → `issues.type` |
| 11 | Unplanned Work % | ❌ NULL | ✅ SQL → `type IN ('BUG','INCIDENT')` |
| 12 | Say/Do Ratio | ❌ NULL | ✅ SQL → `sprint_issues` join |
| 13 | Change Failure Rate | ❌ NULL | ✅ New `convertAlertsToCfr.go` + SQL fix |
| 14 | Code Churn | ⚠️ Wrong formula | ✅ 14-day rework window |
| 15 | Hotfix Deployments | ⚠️ Fragile | ✅ Multi-signal regex |
| 16 | Bugs Introduced | ⚠️ Counts only | ✅ + PR linkage table panel |
