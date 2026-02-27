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

// AiCodeCopilotMetric stores daily GitHub Copilot usage metrics at the org level.
// Sourced from: GET /orgs/{org}/copilot/metrics
// Docs: https://docs.github.com/en/rest/copilot/copilot-metrics
type AiCodeCopilotMetric struct {
	common.NoPKModel

	ConnectionId uint64    `gorm:"primaryKey" json:"connectionId"`
	OrgLogin     string    `gorm:"primaryKey;type:varchar(255)" json:"orgLogin"`
	Date         time.Time `gorm:"primaryKey" json:"date"`

	// Engagement
	TotalActiveUsers   int `json:"totalActiveUsers"`
	TotalEngagedUsers  int `json:"totalEngagedUsers"`

	// IDE Code Completions (aggregate across all languages/editors)
	CompletionEngagedUsers int `json:"completionEngagedUsers"`
	TotalSuggestions       int `json:"totalSuggestions"`
	TotalAcceptances       int `json:"totalAcceptances"`
	TotalLinesSuggested    int `json:"totalLinesSuggested"`
	TotalLinesAccepted     int `json:"totalLinesAccepted"`

	// Chat (IDE + GitHub.com)
	ChatEngagedUsers int `json:"chatEngagedUsers"`
	TotalChats       int `json:"totalChats"`
	TotalChatInsertions int `json:"totalChatInsertions"`
	TotalChatCopies    int `json:"totalChatCopies"`

	// Derived metrics (computed on extract)
	// AcceptanceRate = TotalAcceptances / TotalSuggestions * 100  (%)
	AcceptanceRate float64 `json:"acceptanceRate"`
}

func (AiCodeCopilotMetric) TableName() string {
	return "_tool_aicode_copilot_metrics"
}

// ApiCopilotMetricDay is the raw API response shape from GitHub.
// We use json tags to deserialise; only the fields we need are mapped.
type ApiCopilotMetricDay struct {
	Date              string `json:"date"`
	TotalActiveUsers  int    `json:"total_active_users"`
	TotalEngagedUsers int    `json:"total_engaged_users"`

	CopilotIdeCodeCompletions *struct {
		TotalEngagedUsers int `json:"total_engaged_users"`
		Languages         []struct {
			Name                  string `json:"name"`
			TotalEngagedUsers     int    `json:"total_engaged_users"`
			TotalCodeSuggestions  int    `json:"total_code_suggestions"`
			TotalCodeAcceptances  int    `json:"total_code_acceptances"`
			TotalCodeLinesSuggested int  `json:"total_code_lines_suggested"`
			TotalCodeLinesAccepted  int  `json:"total_code_lines_accepted"`
		} `json:"languages"`
	} `json:"copilot_ide_code_completions"`

	CopilotIdeChat *struct {
		TotalEngagedUsers int `json:"total_engaged_users"`
		Editors           []struct {
			TotalEngagedUsers int `json:"total_engaged_users"`
			TotalChats        int `json:"total_chats"`
			TotalChatInsertions int `json:"total_chat_insertion_events"`
			TotalChatCopies    int `json:"total_chat_copy_events"`
		} `json:"editors"`
	} `json:"copilot_ide_chat"`

	CopilotDotcomChat *struct {
		TotalEngagedUsers int `json:"total_engaged_users"`
		Models            []struct {
			TotalEngagedUsers int `json:"total_engaged_users"`
			TotalChats        int `json:"total_chats"`
		} `json:"models"`
	} `json:"copilot_dotcom_chat"`
}
