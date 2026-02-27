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

package api

import (
	"github.com/apache/incubator-devlake/core/errors"
	coreModels "github.com/apache/incubator-devlake/core/models"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/aicode/models"
	"github.com/apache/incubator-devlake/plugins/aicode/tasks"
)

// MakeDataSourcePipelinePlanV200 builds the pipeline plan for a blueprint scope.
// Each aicode scope (org) becomes one pipeline stage with all enabled subtasks.
func MakeDataSourcePipelinePlanV200(
	subtaskMetas []plugin.SubTaskMeta,
	connectionId uint64,
	bpScopes []*coreModels.BlueprintScope,
) (coreModels.PipelinePlan, []plugin.Scope, errors.Error) {
	scopeDetails, err := dsHelper.ScopeSrv.MapScopeDetails(connectionId, bpScopes)
	if err != nil {
		return nil, nil, err
	}

	plan := make(coreModels.PipelinePlan, 0, len(scopeDetails))
	var scopes []plugin.Scope

	for _, sd := range scopeDetails {
		org, ok := sd.Scope.(*models.AiCodeOrg)
		if !ok {
			return nil, nil, errors.Default.New("unexpected scope type for aicode plugin")
		}

		// Build the options map for this scope.
		options := map[string]interface{}{
			"connectionId": connectionId,
			"orgLogin":     org.Login,
		}
		if sd.ScopeConfig != nil {
			options["scopeConfigId"] = sd.ScopeConfig.(*models.AiCodeScopeConfig).Id
		}

		// Collect the names of enabled subtasks.
		var taskNames []string
		for _, meta := range subtaskMetas {
			if meta.EnabledByDefault {
				taskNames = append(taskNames, meta.Name)
			}
		}

		stage := coreModels.PipelineStage{
			{
				Plugin:   "aicode",
				Subtasks: taskNames,
				Options:  options,
			},
		}
		plan = append(plan, stage)
		scopes = append(scopes, org)
	}

	_ = tasks.CollectCopilotMetricsMeta // keep import used
	return plan, scopes, nil
}
