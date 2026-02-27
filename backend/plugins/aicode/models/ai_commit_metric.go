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

package models

import (
	"time"

	"github.com/apache/incubator-devlake/core/models/common"
)

// AiCodeCommitMetric records per-commit AI attribution, detected by scanning
// commit message trailers (e.g. "Co-authored-by: GitHub Copilot <…>").
// Input data is read from commits already collected by the github/gitextractor plugins.
type AiCodeCommitMetric struct {
	common.NoPKModel

	ConnectionId uint64 `gorm:"primaryKey" json:"connectionId"`
	RepoId       string `gorm:"primaryKey;type:varchar(255)" json:"repoId"`
	CommitSha    string `gorm:"primaryKey;type:varchar(40)" json:"commitSha"`

	// AuthorName / Email copied from the commits table for convenience
	AuthorName  string    `gorm:"type:varchar(255)" json:"authorName"`
	AuthorEmail string    `gorm:"type:varchar(255)" json:"authorEmail"`
	AuthoredAt  time.Time `json:"authoredAt"`

	// AiTool identifies which AI tool wrote (part of) this commit.
	// Values: "copilot", "claude", "other_ai", "" (human)
	AiTool string `gorm:"type:varchar(64)" json:"aiTool"`

	// Additions / Deletions from the commit diff.
	// Populated by the EnrichAiCommitsStats task (reads github_commit_files).
	LinesAdded   int `json:"linesAdded"`
	LinesDeleted int `json:"linesDeleted"`

	// AiGenerated is true when at least one AI trailer was found in the commit message.
	AiGenerated bool `json:"aiGenerated"`

	// TrailerRaw stores the matched trailer line for audit purposes.
	TrailerRaw string `gorm:"type:varchar(512)" json:"trailerRaw,omitempty"`
}

func (AiCodeCommitMetric) TableName() string {
	return "_tool_aicode_commit_metrics"
}

// AiCodeDailyAggregate is a derived/computed table that summarises AI vs human
// lines per day per repo. Used directly for Grafana dashboards.
type AiCodeDailyAggregate struct {
	common.NoPKModel

	ConnectionId uint64    `gorm:"primaryKey" json:"connectionId"`
	RepoId       string    `gorm:"primaryKey;type:varchar(255)" json:"repoId"`
	Date         time.Time `gorm:"primaryKey" json:"date"`

	TotalCommits    int `json:"totalCommits"`
	AiCommits       int `json:"aiCommits"`
	HumanCommits    int `json:"humanCommits"`
	AiLinesAdded    int `json:"aiLinesAdded"`
	HumanLinesAdded int `json:"humanLinesAdded"`
	TotalLinesAdded int `json:"totalLinesAdded"`

	// AiCodeRatio = AiLinesAdded / TotalLinesAdded  (as a percentage 0-100)
	AiCodeRatio float64 `json:"aiCodeRatio"`
}

func (AiCodeDailyAggregate) TableName() string {
	return "_tool_aicode_daily_aggregates"
}
