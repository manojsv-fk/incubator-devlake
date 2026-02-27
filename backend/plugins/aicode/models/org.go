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
	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/plugin"
)

// Ensure AiCodeOrg satisfies the ToolLayerScope interface at compile time.
var _ plugin.ToolLayerScope = (*AiCodeOrg)(nil)

// AiCodeOrg represents a GitHub organization as the unit of collection.
// One org = one scope. All Copilot metrics are gathered at the org level.
type AiCodeOrg struct {
	common.Scope `mapstructure:",squash"`
	// Login is the org slug used in GitHub API paths, e.g. "fourkites"
	Login       string `gorm:"primaryKey;type:varchar(255)" mapstructure:"login" validate:"required" json:"login"`
	Name        string `gorm:"type:varchar(255)" mapstructure:"name" json:"name"`
	Description string `mapstructure:"description,omitempty" json:"description,omitempty"`
	AvatarUrl   string `gorm:"type:varchar(512)" mapstructure:"avatarUrl,omitempty" json:"avatarUrl,omitempty"`
	HtmlUrl     string `gorm:"type:varchar(512)" mapstructure:"htmlUrl,omitempty" json:"htmlUrl,omitempty"`
}

func (AiCodeOrg) TableName() string {
	return "_tool_aicode_orgs"
}

func (o AiCodeOrg) ScopeId() string {
	return o.Login
}

func (o AiCodeOrg) ScopeName() string {
	if o.Name != "" {
		return o.Name
	}
	return o.Login
}

func (o AiCodeOrg) ScopeFullName() string {
	return o.Login
}

func (o AiCodeOrg) ScopeParams() interface{} {
	return &AiCodeApiParams{
		ConnectionId: o.ConnectionId,
		OrgLogin:     o.Login,
	}
}

// AiCodeApiParams is used as the key for raw data tables.
type AiCodeApiParams struct {
	ConnectionId uint64
	OrgLogin     string
}
