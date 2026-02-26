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
	"github.com/apache/incubator-devlake/core/errors"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/chronosphere/models"
)

// ChronosphereOptions contains the pipeline task options.
type ChronosphereOptions struct {
	ConnectionId  uint64                          `json:"connectionId" mapstructure:"connectionId"`
	ScopeConfigId uint64                          `json:"scopeConfigId,omitempty" mapstructure:"scopeConfigId,omitempty"`
	Namespace     string                          `json:"namespace" mapstructure:"namespace" validate:"required"`
	ScopeConfig   *models.ChronosphereScopeConfig `json:"scopeConfig,omitempty" mapstructure:"scopeConfig,omitempty"`
}

// ChronosphereTaskData is available to all sub-tasks.
type ChronosphereTaskData struct {
	Options     *ChronosphereOptions
	ApiClient   *helper.ApiAsyncClient
	Connection  *models.ChronosphereConnection
	ScopeConfig *models.ChronosphereScopeConfig
}

func DecodeTaskOptions(options map[string]interface{}) (*ChronosphereOptions, errors.Error) {
	var op ChronosphereOptions
	if err := helper.Decode(options, &op, nil); err != nil {
		return nil, errors.BadInput.Wrap(err, "could not decode chronosphere task options")
	}
	return &op, nil
}
