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

package impl

import (
	"fmt"

	coreModels "github.com/apache/incubator-devlake/core/models"

	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/aicode/api"
	"github.com/apache/incubator-devlake/plugins/aicode/models"
	"github.com/apache/incubator-devlake/plugins/aicode/models/migrationscripts"
	"github.com/apache/incubator-devlake/plugins/aicode/tasks"
)

// Compile-time check that AiCode implements all required plugin interfaces.
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
} = (*AiCode)(nil)

// AiCode is the top-level plugin struct.
type AiCode struct{}

// ─── PluginMeta ──────────────────────────────────────────────────────────────

func (p AiCode) Name() string {
	return "aicode"
}

func (p AiCode) Description() string {
	return "Collects AI code impact metrics from GitHub Copilot and commit-message trailers (Claude Code, Copilot, etc.)"
}

func (p AiCode) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/aicode"
}

// ─── PluginInit ──────────────────────────────────────────────────────────────

func (p AiCode) Init(basicRes context.BasicRes) errors.Error {
	api.Init(basicRes, p)
	return nil
}

// ─── PluginSource ─────────────────────────────────────────────────────────────

func (p AiCode) Connection() dal.Tabler {
	return &models.AiCodeConnection{}
}

func (p AiCode) Scope() plugin.ToolLayerScope {
	return &models.AiCodeOrg{}
}

func (p AiCode) ScopeConfig() dal.Tabler {
	return &models.AiCodeScopeConfig{}
}

// ─── PluginModel ─────────────────────────────────────────────────────────────

func (p AiCode) GetTablesInfo() []dal.Tabler {
	return []dal.Tabler{
		&models.AiCodeConnection{},
		&models.AiCodeOrg{},
		&models.AiCodeScopeConfig{},
		&models.AiCodeCopilotMetric{},
		&models.AiCodeCommitMetric{},
		&models.AiCodeDailyAggregate{},
	}
}

// ─── PluginMigration ─────────────────────────────────────────────────────────

func (p AiCode) MigrationScripts() []plugin.MigrationScript {
	return migrationscripts.All()
}

// ─── PluginTask ──────────────────────────────────────────────────────────────

func (p AiCode) SubTaskMetas() []plugin.SubTaskMeta {
	return []plugin.SubTaskMeta{
		tasks.CollectCopilotMetricsMeta,
		tasks.ExtractCopilotMetricsMeta,
		tasks.AnalyzeAiCommitsMeta,
	}
}

// PrepareTaskData loads connection + scope config and builds the API client.
func (p AiCode) PrepareTaskData(taskCtx plugin.TaskContext, options map[string]interface{}) (interface{}, errors.Error) {
	op, err := tasks.DecodeTaskOptions(options)
	if err != nil {
		return nil, err
	}

	connection := &models.AiCodeConnection{}
	connHelper := helper.NewConnectionHelper(taskCtx, nil, p.Name())
	if err := connHelper.FirstById(connection, op.ConnectionId); err != nil {
		return nil, err
	}

	apiClient, err := tasks.CreateApiClient(taskCtx, connection)
	if err != nil {
		return nil, err
	}

	// Load scope config if an ID was provided; fall back to defaults.
	var scopeConfig *models.AiCodeScopeConfig
	if op.ScopeConfigId != 0 {
		sc := &models.AiCodeScopeConfig{}
		if dbErr := taskCtx.GetDal().First(sc, dal.Where("id = ?", op.ScopeConfigId)); dbErr == nil {
			scopeConfig = sc
		}
	}
	if scopeConfig == nil {
		defaults := models.DefaultScopeConfig()
		scopeConfig = &defaults
	}
	op.ScopeConfig = scopeConfig

	return &tasks.AiCodeTaskData{
		Options:     op,
		ApiClient:   apiClient,
		Connection:  connection,
		ScopeConfig: scopeConfig,
	}, nil
}

// ─── CloseablePluginTask ─────────────────────────────────────────────────────

func (p AiCode) Close(taskCtx plugin.TaskContext) errors.Error {
	data, ok := taskCtx.GetData().(*tasks.AiCodeTaskData)
	if !ok {
		return errors.Default.New(fmt.Sprintf("GetData failed for aicode close: %+v", taskCtx))
	}
	data.ApiClient.Release()
	return nil
}

// ─── DataSourcePluginBlueprintV200 ───────────────────────────────────────────

func (p AiCode) MakeDataSourcePipelinePlanV200(
	connectionId uint64,
	scopes []*coreModels.BlueprintScope,
) (pp coreModels.PipelinePlan, sc []plugin.Scope, err errors.Error) {
	return api.MakeDataSourcePipelinePlanV200(p.SubTaskMetas(), connectionId, scopes)
}

// ─── PluginApi ───────────────────────────────────────────────────────────────

func (p AiCode) ApiResources() map[string]map[string]plugin.ApiResourceHandler {
	return map[string]map[string]plugin.ApiResourceHandler{
		"test": {
			"POST": api.TestConnection,
		},
		"connections": {
			"POST": api.PostConnections,
			"GET":  api.ListConnections,
		},
		"connections/:connectionId": {
			"GET":    api.GetConnection,
			"PATCH":  api.PatchConnection,
			"DELETE": api.DeleteConnection,
		},
		"connections/:connectionId/test": {
			"POST": api.TestExistingConnection,
		},
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
		"scope-config/:scopeConfigId/projects": {
			"GET": api.GetProjectsByScopeConfig,
		},
	}
}
