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
	"encoding/json"
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/aicode/models"
)

// ExtractCopilotMetricsMeta is the SubTaskMeta for the extractor.
var ExtractCopilotMetricsMeta = plugin.SubTaskMeta{
	Name:             "extractCopilotMetrics",
	EntryPoint:       ExtractCopilotMetrics,
	EnabledByDefault: true,
	Description:      "Parse raw Copilot API responses into tool-layer table _tool_aicode_copilot_metrics",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

// ExtractCopilotMetrics reads raw JSON rows from _raw_aicode_api_copilot_metrics
// and writes structured records to _tool_aicode_copilot_metrics.
func ExtractCopilotMetrics(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*AiCodeTaskData)

	extractor, err := helper.NewApiExtractor(helper.ApiExtractorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: models.AiCodeApiParams{
				ConnectionId: data.Options.ConnectionId,
				OrgLogin:     data.Options.OrgLogin,
			},
			Table: RAW_COPILOT_METRICS_TABLE,
		},
		Extract: func(row *helper.RawData) ([]interface{}, errors.Error) {
			var apiDay models.ApiCopilotMetricDay
			if err := errors.Convert(json.Unmarshal(row.Data, &apiDay)); err != nil {
				return nil, err
			}

			date, parseErr := time.Parse("2006-01-02", apiDay.Date)
			if parseErr != nil {
				return nil, errors.Convert(parseErr)
			}

			metric := &models.AiCodeCopilotMetric{
				ConnectionId:      data.Options.ConnectionId,
				OrgLogin:          data.Options.OrgLogin,
				Date:              date,
				TotalActiveUsers:  apiDay.TotalActiveUsers,
				TotalEngagedUsers: apiDay.TotalEngagedUsers,
			}

			// Aggregate completions across all languages
			if cc := apiDay.CopilotIdeCodeCompletions; cc != nil {
				metric.CompletionEngagedUsers = cc.TotalEngagedUsers
				for _, lang := range cc.Languages {
					metric.TotalSuggestions += lang.TotalCodeSuggestions
					metric.TotalAcceptances += lang.TotalCodeAcceptances
					metric.TotalLinesSuggested += lang.TotalCodeLinesSuggested
					metric.TotalLinesAccepted += lang.TotalCodeLinesAccepted
				}
				if metric.TotalSuggestions > 0 {
					metric.AcceptanceRate = float64(metric.TotalAcceptances) / float64(metric.TotalSuggestions) * 100
				}
			}

			// Aggregate IDE chat
			if chat := apiDay.CopilotIdeChat; chat != nil {
				metric.ChatEngagedUsers = chat.TotalEngagedUsers
				for _, ed := range chat.Editors {
					metric.TotalChats += ed.TotalChats
					metric.TotalChatInsertions += ed.TotalChatInsertions
					metric.TotalChatCopies += ed.TotalChatCopies
				}
			}

			// Add dotcom chat on top
			if dotcom := apiDay.CopilotDotcomChat; dotcom != nil {
				metric.ChatEngagedUsers += dotcom.TotalEngagedUsers
				for _, m := range dotcom.Models {
					metric.TotalChats += m.TotalChats
				}
			}

			return []interface{}{metric}, nil
		},
	})
	if err != nil {
		return err
	}

	return extractor.Execute()
}
