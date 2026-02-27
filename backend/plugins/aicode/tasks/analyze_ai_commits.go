/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tasks

import (
	"regexp"
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/domainlayer/code"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/aicode/models"
)

// AnalyzeAiCommitsMeta is the SubTaskMeta for the AI commit analyser.
var AnalyzeAiCommitsMeta = plugin.SubTaskMeta{
	Name:             "analyzeAiCommits",
	EntryPoint:       AnalyzeAiCommits,
	EnabledByDefault: true,
	Description:      "Scan commits for AI attribution trailers and write per-commit AI metrics",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
	// Depends on commits being collected first (by github / gitextractor plugins).
	DependencyTables: []string{"commits", "repo_commits"},
	ProductTables:    []string{"_tool_aicode_commit_metrics", "_tool_aicode_daily_aggregates"},
}

// commitRow is a lightweight struct used for the DB cursor query.
type commitRow struct {
	Sha          string
	Message      string
	AuthorName   string
	AuthorEmail  string
	AuthoredDate time.Time
	Additions    int
	Deletions    int
	RepoId       string
}

// AnalyzeAiCommits reads commits from DevLake's domain-layer `commits` table
// (populated by the github/gitextractor plugins) and checks each commit's
// message for AI attribution trailers.  Results are written to
// _tool_aicode_commit_metrics and _tool_aicode_daily_aggregates.
func AnalyzeAiCommits(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*AiCodeTaskData)
	db := taskCtx.GetDal()
	logger := taskCtx.GetLogger()

	// Compile trailer regexes from scope config (fall back to defaults).
	sc := data.ScopeConfig
	if sc == nil {
		defaults := models.DefaultScopeConfig()
		sc = &defaults
	}

	copilotRe, err := compilePattern(sc.CopilotTrailerPattern, `(?i)co-authored-by:.*copilot`)
	if err != nil {
		return errors.BadInput.Wrap(err, "invalid copilotTrailerPattern")
	}
	claudeRe, err := compilePattern(sc.ClaudeTrailerPattern, `(?i)co-authored-by:.*claude`)
	if err != nil {
		return errors.BadInput.Wrap(err, "invalid claudeTrailerPattern")
	}
	genericAiRe, err := compilePattern(sc.GenericAiTrailerPattern, `(?i)(generated-by|co-authored-by):.*\b(ai|bot)\b`)
	if err != nil {
		return errors.BadInput.Wrap(err, "invalid genericAiTrailerPattern")
	}

	// Query the domain-layer commits table joined with repo_commits so we know
	// which repos belong to this org connection.
	// The repo_id in repo_commits looks like "github:GithubRepo:1:fourkites/someRepo".
	orgPrefix := "github:GithubRepo:" + uint64ToStr(data.Options.ConnectionId) + ":"

	cursor, dbErr := db.Cursor(
		dal.Select("c.sha, c.message, c.author_name, c.author_email, c.authored_date, c.additions, c.deletions, rc.repo_id"),
		dal.From("commits c"),
		dal.Join("LEFT JOIN repo_commits rc ON rc.commit_sha = c.sha"),
		dal.Where("rc.repo_id LIKE ?", orgPrefix+"%"),
	)
	if dbErr != nil {
		return dbErr
	}
	defer cursor.Close()

	// dailyMap accumulates per-day, per-repo aggregates before final upsert.
	type dayRepoKey struct {
		repoId string
		date   string // YYYY-MM-DD
	}
	dailyMap := make(map[dayRepoKey]*models.AiCodeDailyAggregate)

	for cursor.Next() {
		row := &commitRow{}
		if scanErr := db.Fetch(cursor, row); scanErr != nil {
			return scanErr
		}

		aiTool, trailerRaw := detectAiTrailer(row.Message, copilotRe, claudeRe, genericAiRe)
		isAi := aiTool != ""

		metric := &models.AiCodeCommitMetric{
			ConnectionId: data.Options.ConnectionId,
			RepoId:       row.RepoId,
			CommitSha:    row.Sha,
			AuthorName:   row.AuthorName,
			AuthorEmail:  row.AuthorEmail,
			AuthoredAt:   row.AuthoredDate,
			AiTool:       aiTool,
			LinesAdded:   row.Additions,
			LinesDeleted: row.Deletions,
			AiGenerated:  isAi,
			TrailerRaw:   trailerRaw,
		}

		if dbErr := db.CreateOrUpdate(metric); dbErr != nil {
			logger.Warn(dbErr, "failed to upsert AiCodeCommitMetric for sha %s", row.Sha)
		}

		// Accumulate daily aggregates
		dateStr := row.AuthoredDate.UTC().Format("2006-01-02")
		key := dayRepoKey{repoId: row.RepoId, date: dateStr}
		agg, ok := dailyMap[key]
		if !ok {
			t, _ := time.Parse("2006-01-02", dateStr)
			agg = &models.AiCodeDailyAggregate{
				ConnectionId: data.Options.ConnectionId,
				RepoId:       row.RepoId,
				Date:         t,
			}
			dailyMap[key] = agg
		}
		agg.TotalCommits++
		agg.TotalLinesAdded += row.Additions
		if isAi {
			agg.AiCommits++
			agg.AiLinesAdded += row.Additions
		} else {
			agg.HumanCommits++
			agg.HumanLinesAdded += row.Additions
		}
	}

	// Write daily aggregates
	for _, agg := range dailyMap {
		if agg.TotalLinesAdded > 0 {
			agg.AiCodeRatio = float64(agg.AiLinesAdded) / float64(agg.TotalLinesAdded) * 100
		}
		if dbErr := db.CreateOrUpdate(agg); dbErr != nil {
			logger.Warn(dbErr, "failed to upsert AiCodeDailyAggregate for repo %s date %s", agg.RepoId, agg.Date)
		}
	}

	logger.Info("analyzed %d commit records for AI trailers", len(dailyMap))
	return nil
}

// detectAiTrailer checks all lines in the commit message against the known AI
// trailer patterns. Returns (tool name, matched line) or ("", "") if none found.
func detectAiTrailer(message string, copilotRe, claudeRe, genericRe *regexp.Regexp) (string, string) {
	for _, line := range strings.Split(message, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if copilotRe.MatchString(line) {
			return "copilot", line
		}
		if claudeRe.MatchString(line) {
			return "claude", line
		}
		if genericRe.MatchString(line) {
			return "other_ai", line
		}
	}
	return "", ""
}

// compilePattern compiles a regex pattern, falling back to the provided default
// if the user-supplied pattern is empty.
func compilePattern(pattern, fallback string) (*regexp.Regexp, error) {
	p := strings.TrimSpace(pattern)
	if p == "" {
		p = fallback
	}
	return regexp.Compile(p)
}

// uint64ToStr converts a uint64 to its decimal string representation without
// importing strconv (avoids import cycle risk in this package).
func uint64ToStr(n uint64) string {
	if n == 0 {
		return "0"
	}
	digits := make([]byte, 0, 20)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// Compile-time check that code.Commit exists (we depend on that table).
var _ = (*code.Commit)(nil)
