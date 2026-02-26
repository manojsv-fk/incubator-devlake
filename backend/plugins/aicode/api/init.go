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
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/aicode/models"
	"github.com/go-playground/validator/v10"
)

var vld *validator.Validate
var basicRes context.BasicRes

// dsHelper centralises CRUD operations for connections, orgs (scopes), and scope configs.
var dsHelper *helper.DsHelper[models.AiCodeConnection, models.AiCodeOrg, models.AiCodeScopeConfig]
var raProxy *helper.DsRemoteApiProxyHelper[models.AiCodeConnection]

func Init(br context.BasicRes, p plugin.PluginMeta) {
	basicRes = br
	vld = validator.New()

	dsHelper = helper.NewDataSourceHelper[
		models.AiCodeConnection,
		models.AiCodeOrg,
		models.AiCodeScopeConfig,
	](
		br,
		p.Name(),
		[]string{"login"},
		func(c models.AiCodeConnection) models.AiCodeConnection {
			return c.Sanitize()
		},
		nil,
		nil,
	)

	raProxy = helper.NewDsRemoteApiProxyHelper[models.AiCodeConnection](dsHelper.ConnApi.ModelApiHelper)
}
