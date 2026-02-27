/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements. See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License. You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package tasks contains the data collection subtasks for the Chronosphere plugin.
//
// Chronosphere API reference: https://docs.chronosphere.io/api
//
// Key endpoints used:
//   - GET /api/v1/events             – alert firing/resolution events
//   - GET /api/v1/monitor/monitors   – monitor (alert rule) definitions
//   - GET /api/v1/rollups/query      – metric rollup queries for SLO data
//
// Authentication: Bearer <API token>  (X-Auth-Token header)
package tasks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/chronosphere/models"
)

const RAW_ALERT_EVENTS_TABLE = "chronosphere_api_alert_events"

// CollectAlertEventsMeta registers the collector subtask.
var CollectAlertEventsMeta = plugin.SubTaskMeta{
	Name:             "collectAlertEvents",
	EntryPoint:       CollectAlertEvents,
	EnabledByDefault: true,
	Description:      "Collect alert firing/resolution events from Chronosphere API for MTTR and Change Failure Rate",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CICD},
}

// CollectAlertEvents fetches alert events from Chronosphere.
// Endpoint: GET /api/v1/events
// Supports time-range filtering via `start_time` and `end_time` query params.
func CollectAlertEvents(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*ChronosphereTaskData)

	collector, err := helper.NewApiCollector(helper.ApiCollectorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: models.ChronosphereApiParams{
				ConnectionId: data.Options.ConnectionId,
				Namespace:    data.Options.Namespace,
			},
			Table: RAW_ALERT_EVENTS_TABLE,
		},
		ApiClient:   data.ApiClient,
		UrlTemplate: "api/v1/events",
		PageSize:    200,
		Query: func(reqData *helper.RequestData) (url.Values, errors.Error) {
			q := url.Values{}
			// Fetch last 30 days by default; the timeAfter parameter will be
			// respected if set via the blueprint's incremental sync.
			q.Set("namespace", data.Options.Namespace)
			q.Set("page_size", fmt.Sprintf("%d", reqData.Pager.Size))
			q.Set("page_token", fmt.Sprintf("%d", reqData.Pager.Skip))
			// Chronosphere uses RFC3339 for time bounds.
			endTime := time.Now().UTC().Format(time.RFC3339)
			startTime := time.Now().UTC().AddDate(0, 0, -30).Format(time.RFC3339)
			q.Set("start_time", startTime)
			q.Set("end_time", endTime)
			return q, nil
		},
		ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			// Chronosphere response shape:
			// { "events": [ {...}, ... ], "next_page_token": "..." }
			var body struct {
				Events        []json.RawMessage `json:"events"`
				NextPageToken string            `json:"next_page_token"`
			}
			if err := helper.UnmarshalResponse(res, &body); err != nil {
				return nil, err
			}
			return body.Events, nil
		},
	})
	if err != nil {
		return err
	}

	return collector.Execute()
}

// ─── Extract ─────────────────────────────────────────────────────────────────

const RAW_INCIDENTS_TABLE = "chronosphere_api_incidents"

// ExtractAlertEventsMeta registers the extractor subtask.
var ExtractAlertEventsMeta = plugin.SubTaskMeta{
	Name:             "extractAlertEvents",
	EntryPoint:       ExtractAlertEvents,
	EnabledByDefault: true,
	Description:      "Parse raw Chronosphere alert events into _tool_chronosphere_alert_events",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CICD},
}

// apiAlertEvent mirrors the Chronosphere event JSON shape.
type apiAlertEvent struct {
	ID         string     `json:"id"`
	AlertName  string     `json:"monitor_name"`
	Namespace  string     `json:"namespace"`
	Severity   string     `json:"severity"`
	Status     string     `json:"status"` // "firing" or "resolved"
	FiredAt    *time.Time `json:"started_at"`
	ResolvedAt *time.Time `json:"ended_at"`
	Labels     string     `json:"labels"` // JSON object
}

// ExtractAlertEvents reads the raw table and writes to _tool_chronosphere_alert_events.
func ExtractAlertEvents(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*ChronosphereTaskData)

	extractor, err := helper.NewApiExtractor(helper.ApiExtractorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: models.ChronosphereApiParams{
				ConnectionId: data.Options.ConnectionId,
				Namespace:    data.Options.Namespace,
			},
			Table: RAW_ALERT_EVENTS_TABLE,
		},
		Extract: func(row *helper.RawData) ([]interface{}, errors.Error) {
			var apiEvent apiAlertEvent
			if err := errors.Convert(json.Unmarshal(row.Data, &apiEvent)); err != nil {
				return nil, err
			}

			event := &models.ChronosphereAlertEvent{
				ConnectionId: data.Options.ConnectionId,
				Namespace:    data.Options.Namespace,
				EventId:      apiEvent.ID,
				AlertName:    apiEvent.AlertName,
				Severity:     apiEvent.Severity,
				Status:       apiEvent.Status,
				FiredAt:      apiEvent.FiredAt,
				ResolvedAt:   apiEvent.ResolvedAt,
				Labels:       apiEvent.Labels,
			}

			if event.FiredAt != nil && event.ResolvedAt != nil {
				event.DurationMinutes = event.ResolvedAt.Sub(*event.FiredAt).Minutes()
			}

			return []interface{}{event}, nil
		},
	})
	if err != nil {
		return err
	}

	return extractor.Execute()
}
