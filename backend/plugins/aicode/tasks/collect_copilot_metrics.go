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
	"fmt"
	"net/http"
	"net/url"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/aicode/models"
)

const RAW_COPILOT_METRICS_TABLE = "aicode_api_copilot_metrics"

// CollectCopilotMetricsMeta is the SubTaskMeta for the collector.
var CollectCopilotMetricsMeta = plugin.SubTaskMeta{
	Name:             "collectCopilotMetrics",
	EntryPoint:       CollectCopilotMetrics,
	EnabledByDefault: true,
	Description:      "Collect daily GitHub Copilot metrics from the Org Copilot Metrics API",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

// CollectCopilotMetrics fetches daily Copilot usage from:
// GET /orgs/{org}/copilot/metrics
// The API returns up to 28 days of history.
// Requires PAT scope: manage_billing:copilot OR read:org
func CollectCopilotMetrics(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*AiCodeTaskData)

	collector, err := helper.NewApiCollector(helper.ApiCollectorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: models.AiCodeApiParams{
				ConnectionId: data.Options.ConnectionId,
				OrgLogin:     data.Options.OrgLogin,
			},
			Table: RAW_COPILOT_METRICS_TABLE,
		},
		ApiClient:   data.ApiClient,
		UrlTemplate: fmt.Sprintf("orgs/%s/copilot/metrics", data.Options.OrgLogin),
		Query: func(reqData *helper.RequestData) (url.Values, errors.Error) {
			// The API accepts optional `since` and `until` query params (YYYY-MM-DD).
			// Without them it returns the last 28 days.
			return nil, nil
		},
		ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			// The response is a top-level JSON array of daily metric objects.
			var items []json.RawMessage
			if err := helper.UnmarshalResponse(res, &items); err != nil {
				return nil, err
			}
			return items, nil
		},
	})
	if err != nil {
		return err
	}

	return collector.Execute()
}
