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

package api

import (
	"github.com/apache/incubator-devlake/core/errors"
	coreModels "github.com/apache/incubator-devlake/core/models"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/chronosphere/models"
	"github.com/apache/incubator-devlake/plugins/chronosphere/tasks"
)

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
		ns, ok := sd.Scope.(*models.ChronosphereNamespace)
		if !ok {
			return nil, nil, errors.Default.New("unexpected scope type for chronosphere plugin")
		}

		options := map[string]interface{}{
			"connectionId": connectionId,
			"namespace":    ns.Namespace,
		}

		var taskNames []string
		for _, meta := range subtaskMetas {
			if meta.EnabledByDefault {
				taskNames = append(taskNames, meta.Name)
			}
		}

		plan = append(plan, coreModels.PipelineStage{{
			Plugin:   "chronosphere",
			Subtasks: taskNames,
			Options:  options,
		}})
		scopes = append(scopes, ns)
	}

	_ = tasks.CollectAlertEventsMeta // keep import
	return plan, scopes, nil
}
