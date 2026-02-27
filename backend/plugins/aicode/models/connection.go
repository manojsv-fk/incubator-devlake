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

package models

import (
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

// AiCodeConn holds the credentials needed to connect to the GitHub API
// for Copilot metrics. Uses a Personal Access Token with:
//   - manage_billing:copilot OR read:org scope (for Copilot Metrics API)
//   - repo scope (for commit trailer analysis)
type AiCodeConn struct {
	helper.RestConnection `mapstructure:",squash"`
	helper.AccessToken    `mapstructure:",squash"`
}

func (c AiCodeConn) Sanitize() AiCodeConn {
	c.Token = ""
	return c
}

// AiCodeConnection is stored in the database.
// It holds the GitHub PAT and the target GitHub organization.
type AiCodeConnection struct {
	helper.BaseConnection `mapstructure:",squash"`
	AiCodeConn            `mapstructure:",squash"`
	// OrgLogin is the GitHub organization slug, e.g. "fourkites"
	OrgLogin string `json:"orgLogin" gorm:"type:varchar(255)" mapstructure:"orgLogin" validate:"required"`
}

func (AiCodeConnection) TableName() string {
	return "_tool_aicode_connections"
}

func (c AiCodeConnection) Sanitize() AiCodeConnection {
	c.AiCodeConn = c.AiCodeConn.Sanitize()
	return c
}

func (c *AiCodeConnection) MergeFromRequest(target *AiCodeConnection, body map[string]interface{}) error {
	savedToken := target.Token
	if err := helper.DecodeMapStruct(body, target, true); err != nil {
		return err
	}
	// preserve token if not provided in the update body
	if target.Token == "" {
		target.Token = savedToken
	}
	return nil
}
