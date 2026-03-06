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

package tasks

import (
	"fmt"
	"reflect"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/domainlayer"
	"github.com/apache/incubator-devlake/core/models/domainlayer/ticket"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/chronosphere/models"
)

// ConvertAlertsToCfrMeta registers the converter subtask that promotes resolved
// Chronosphere alert events into the domain `incidents` table so DORA's
// ConnectIncidentToDeployment task can compute Change Failure Rate.
var ConvertAlertsToCfrMeta = plugin.SubTaskMeta{
	Name:             "convertAlertsToCfr",
	EntryPoint:       ConvertAlertsToCfr,
	EnabledByDefault: true,
	Description:      "Convert resolved Chronosphere alert events into domain incidents for DORA Change Failure Rate",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CICD},
}

// ConvertAlertsToCfr reads _tool_chronosphere_alert_events (resolved events)
// and writes a ticket.Incident row per event so the DORA plugin can join them
// to deployments via project_incident_deployment_relationships.
func ConvertAlertsToCfr(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*ChronosphereTaskData)

	cursor, err := db.Cursor(
		dal.From(&models.ChronosphereAlertEvent{}),
		dal.Where(
			"connection_id = ? AND namespace = ? AND resolved_at IS NOT NULL",
			data.Options.ConnectionId,
			data.Options.Namespace,
		),
	)
	if err != nil {
		return err
	}
	defer cursor.Close()

	converter, err := helper.NewDataConverter(helper.DataConverterArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: models.ChronosphereApiParams{
				ConnectionId: data.Options.ConnectionId,
				Namespace:    data.Options.Namespace,
			},
			Table: RAW_ALERT_EVENTS_TABLE,
		},
		InputRowType: reflect.TypeOf(models.ChronosphereAlertEvent{}),
		Input:        cursor,
		Convert: func(inputRow interface{}) ([]interface{}, errors.Error) {
			event := inputRow.(*models.ChronosphereAlertEvent)

			// Skip events that never fired (guard against bad data).
			if event.FiredAt == nil {
				return nil, nil
			}

			domainId := fmt.Sprintf(
				"chronosphere:ChronosphereAlertEvent:%d:%s:%s",
				event.ConnectionId,
				event.Namespace,
				event.EventId,
			)

			status := ticket.IN_PROGRESS
			if event.ResolvedAt != nil {
				status = ticket.DONE
			}

			incident := &ticket.Incident{
				DomainEntity: domainlayer.DomainEntity{Id: domainId},
				Title:        event.AlertName,
				Status:       status,
				OriginalStatus: event.Status,
				Severity:     event.Severity,
				CreatedDate:  event.FiredAt,
				ResolutionDate: event.ResolvedAt,
				// Table + ScopeId are used by DORA's ConnectIncidentToDeployment
				// to join via project_mapping.
				Table:   models.ChronosphereNamespace{}.TableName(),
				ScopeId: event.Namespace,
			}

			if event.FiredAt != nil && event.ResolvedAt != nil {
				mins := uint(event.ResolvedAt.Sub(*event.FiredAt).Minutes())
				incident.LeadTimeMinutes = &mins
			}

			return []interface{}{incident}, nil
		},
	})
	if err != nil {
		return err
	}

	return converter.Execute()
}
