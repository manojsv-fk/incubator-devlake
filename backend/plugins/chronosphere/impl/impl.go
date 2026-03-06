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

package impl

import (
	"fmt"

	coreModels "github.com/apache/incubator-devlake/core/models"

	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/chronosphere/api"
	"github.com/apache/incubator-devlake/plugins/chronosphere/models"
	"github.com/apache/incubator-devlake/plugins/chronosphere/models/migrationscripts"
	"github.com/apache/incubator-devlake/plugins/chronosphere/tasks"
)

var _ interface {
	plugin.PluginMeta
	plugin.PluginInit
	plugin.PluginTask
	plugin.PluginApi
	plugin.PluginModel
	plugin.PluginMigration
	plugin.CloseablePluginTask
	plugin.PluginSource
	plugin.DataSourcePluginBlueprintV200
} = (*Chronosphere)(nil)

type Chronosphere struct{}

func (p Chronosphere) Name() string        { return "chronosphere" }
func (p Chronosphere) Description() string { return "Collect alert events from Chronosphere for MTTR and Change Failure Rate (DORA metrics)" }
func (p Chronosphere) RootPkgPath() string { return "github.com/apache/incubator-devlake/plugins/chronosphere" }

func (p Chronosphere) Init(basicRes context.BasicRes) errors.Error {
	api.Init(basicRes, p)
	return nil
}

func (p Chronosphere) Connection() dal.Tabler  { return &models.ChronosphereConnection{} }
func (p Chronosphere) Scope() plugin.ToolLayerScope { return &models.ChronosphereNamespace{} }
func (p Chronosphere) ScopeConfig() dal.Tabler { return &models.ChronosphereScopeConfig{} }

func (p Chronosphere) GetTablesInfo() []dal.Tabler {
	return []dal.Tabler{
		&models.ChronosphereConnection{},
		&models.ChronosphereNamespace{},
		&models.ChronosphereScopeConfig{},
		&models.ChronosphereAlertEvent{},
		&models.ChronosphereIncident{},
	}
}

func (p Chronosphere) MigrationScripts() []plugin.MigrationScript {
	return migrationscripts.All()
}

func (p Chronosphere) SubTaskMetas() []plugin.SubTaskMeta {
	return []plugin.SubTaskMeta{
		tasks.CollectAlertEventsMeta,
		tasks.ExtractAlertEventsMeta,
		tasks.ConvertAlertsToCfrMeta,
	}
}

func (p Chronosphere) PrepareTaskData(taskCtx plugin.TaskContext, options map[string]interface{}) (interface{}, errors.Error) {
	op, err := tasks.DecodeTaskOptions(options)
	if err != nil {
		return nil, err
	}

	connection := &models.ChronosphereConnection{}
	connHelper := helper.NewConnectionHelper(taskCtx, nil, p.Name())
	if err := connHelper.FirstById(connection, op.ConnectionId); err != nil {
		return nil, err
	}

	apiClient, err := createApiClient(taskCtx, connection)
	if err != nil {
		return nil, err
	}

	return &tasks.ChronosphereTaskData{
		Options:    op,
		ApiClient:  apiClient,
		Connection: connection,
	}, nil
}

// createApiClient builds a Bearer-token authenticated client for Chronosphere.
func createApiClient(taskCtx plugin.TaskContext, conn *models.ChronosphereConnection) (*helper.ApiAsyncClient, errors.Error) {
	apiClient, err := helper.NewApiClientFromConnection(taskCtx.GetContext(), taskCtx, &conn.ChronosphereConn)
	if err != nil {
		return nil, err
	}
	apiClient.SetHeaders(map[string]string{
		"X-Auth-Token": conn.Token,
		"Accept":       "application/json",
	})
	return helper.CreateAsyncApiClient(taskCtx, apiClient, nil)
}

func (p Chronosphere) Close(taskCtx plugin.TaskContext) errors.Error {
	data, ok := taskCtx.GetData().(*tasks.ChronosphereTaskData)
	if !ok {
		return errors.Default.New(fmt.Sprintf("GetData failed for chronosphere close: %+v", taskCtx))
	}
	data.ApiClient.Release()
	return nil
}

func (p Chronosphere) MakeDataSourcePipelinePlanV200(
	connectionId uint64,
	scopes []*coreModels.BlueprintScope,
) (coreModels.PipelinePlan, []plugin.Scope, errors.Error) {
	return api.MakeDataSourcePipelinePlanV200(p.SubTaskMetas(), connectionId, scopes)
}

func (p Chronosphere) ApiResources() map[string]map[string]plugin.ApiResourceHandler {
	return map[string]map[string]plugin.ApiResourceHandler{
		"test": {"POST": api.TestConnection},
		"connections": {
			"POST": api.PostConnections,
			"GET":  api.ListConnections,
		},
		"connections/:connectionId": {
			"GET":    api.GetConnection,
			"PATCH":  api.PatchConnection,
			"DELETE": api.DeleteConnection,
		},
		"connections/:connectionId/test":  {"POST": api.TestExistingConnection},
		"connections/:connectionId/scopes": {
			"GET": api.GetScopeList,
			"PUT": api.PutScopes,
		},
		"connections/:connectionId/scopes/:scopeId": {
			"GET":    api.GetScope,
			"PATCH":  api.PatchScope,
			"DELETE": api.DeleteScope,
		},
		"connections/:connectionId/scope-configs": {
			"POST": api.CreateScopeConfig,
			"GET":  api.GetScopeConfigList,
		},
		"connections/:connectionId/scope-configs/:scopeConfigId": {
			"GET":    api.GetScopeConfig,
			"PATCH":  api.UpdateScopeConfig,
			"DELETE": api.DeleteScopeConfig,
		},
	}
}
