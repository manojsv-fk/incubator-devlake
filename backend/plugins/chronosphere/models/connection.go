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

// Package models contains the data models for the chronosphere DevLake plugin.
// Chronosphere is FourKites' observability platform (MTTR / Change Failure Rate).
// API docs: https://docs.chronosphere.io/api
package models

import (
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

// ChronosphereConn holds credentials for the Chronosphere API.
// Authentication: Bearer token (Settings → API Tokens in Chronosphere UI).
type ChronosphereConn struct {
	helper.RestConnection `mapstructure:",squash"`
	helper.AccessToken    `mapstructure:",squash"`
	// OrgSlug is the organisation slug visible in the Chronosphere URL.
	OrgSlug string `json:"orgSlug" gorm:"type:varchar(255)" mapstructure:"orgSlug"`
}

func (c ChronosphereConn) Sanitize() ChronosphereConn {
	c.Token = ""
	return c
}

// ChronosphereConnection is persisted to the DevLake database.
type ChronosphereConnection struct {
	helper.BaseConnection `mapstructure:",squash"`
	ChronosphereConn      `mapstructure:",squash"`
}

func (ChronosphereConnection) TableName() string {
	return "_tool_chronosphere_connections"
}

func (c ChronosphereConnection) Sanitize() ChronosphereConnection {
	c.ChronosphereConn = c.ChronosphereConn.Sanitize()
	return c
}

func (c *ChronosphereConnection) MergeFromRequest(target *ChronosphereConnection, body map[string]interface{}) error {
	savedToken := target.Token
	if err := helper.DecodeMapStruct(body, target, true); err != nil {
		return err
	}
	if target.Token == "" {
		target.Token = savedToken
	}
	return nil
}
