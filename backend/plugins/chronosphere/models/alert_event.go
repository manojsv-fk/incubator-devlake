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
	"time"

	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/models/domainlayer/devops"
)

// ChronosphereAlertEvent stores individual alert firing/resolution events.
// These events are the raw input for computing MTTR and Change Failure Rate.
//
// Data source: Chronosphere Event API
// Endpoint: GET /api/v1/events  (filter by namespace and time range)
type ChronosphereAlertEvent struct {
	common.NoPKModel

	ConnectionId uint64 `gorm:"primaryKey" json:"connectionId"`
	Namespace    string `gorm:"primaryKey;type:varchar(255)" json:"namespace"`
	// EventId is Chronosphere's unique event identifier.
	EventId string `gorm:"primaryKey;type:varchar(255)" json:"eventId"`

	// AlertName is the name of the Chronosphere alert that fired.
	AlertName string `gorm:"type:varchar(512)" json:"alertName"`
	// Severity maps to Chronosphere severity levels: critical, high, medium, low.
	Severity string `gorm:"type:varchar(64)" json:"severity"`

	// Status: "firing" when alert starts, "resolved" when it clears.
	Status string `gorm:"type:varchar(32)" json:"status"`

	FiredAt    *time.Time `json:"firedAt"`
	ResolvedAt *time.Time `json:"resolvedAt,omitempty"`

	// DurationMinutes is populated on resolution (ResolvedAt - FiredAt).
	DurationMinutes float64 `json:"durationMinutes,omitempty"`

	// IsChangeFailure is true when the alert fired within the post-deploy
	// observation window (configurable, default 1 hour after a deployment).
	// Computed by ConvertAlertEvents task.
	IsChangeFailure bool `json:"isChangeFailure"`

	// LinkedDeploymentId links to a Jenkins build in the cicd_tasks table.
	LinkedDeploymentId string `gorm:"type:varchar(255)" json:"linkedDeploymentId,omitempty"`

	Labels string `gorm:"type:text" json:"labels,omitempty"` // JSON-encoded label map
}

func (ChronosphereAlertEvent) TableName() string {
	return "_tool_chronosphere_alert_events"
}

// ChronosphereIncident aggregates related alert events into a single incident
// for MTTR calculation.
type ChronosphereIncident struct {
	common.NoPKModel

	ConnectionId string `gorm:"primaryKey" json:"connectionId"`
	Namespace    string `gorm:"primaryKey;type:varchar(255)" json:"namespace"`
	IncidentId   string `gorm:"primaryKey;type:varchar(255)" json:"incidentId"`

	Title       string     `gorm:"type:varchar(512)" json:"title"`
	Severity    string     `gorm:"type:varchar(64)" json:"severity"`
	StartedAt   time.Time  `json:"startedAt"`
	ResolvedAt  *time.Time `json:"resolvedAt,omitempty"`
	MTTRMinutes float64    `json:"mttrMinutes,omitempty"`
	Resolved    bool       `json:"resolved"`
}

func (ChronosphereIncident) TableName() string {
	return "_tool_chronosphere_incidents"
}

// ChronosphereNamespace is the scope (collection unit) for Chronosphere.
type ChronosphereNamespace struct {
	common.Scope `mapstructure:",squash"`
	Namespace    string `gorm:"primaryKey;type:varchar(255)" mapstructure:"namespace" validate:"required" json:"namespace"`
	DisplayName  string `gorm:"type:varchar(255)" json:"displayName,omitempty"`
}

func (ChronosphereNamespace) TableName() string {
	return "_tool_chronosphere_namespaces"
}

func (n ChronosphereNamespace) ScopeId() string   { return n.Namespace }
func (n ChronosphereNamespace) ScopeName() string  { return n.DisplayName }
func (n ChronosphereNamespace) ScopeFullName() string { return n.Namespace }
func (n ChronosphereNamespace) ScopeParams() interface{} {
	return &ChronosphereApiParams{
		ConnectionId: n.ConnectionId,
		Namespace:    n.Namespace,
	}
}

// ChronosphereScopeConfig holds transformation rules for Chronosphere data.
type ChronosphereScopeConfig struct {
	common.ScopeConfig `mapstructure:",squash" json:",inline" gorm:"embedded"`

	// ProductionAlertPattern is a regex to filter which alerts contribute
	// to Change Failure Rate (e.g. only production severity alerts).
	ProductionAlertPattern string `gorm:"type:varchar(512)" json:"productionAlertPattern,omitempty"`

	// ChangeFailureWindowMinutes is how long after a deployment we consider
	// an alert to be a "change failure". Default: 60.
	ChangeFailureWindowMinutes int `gorm:"default:60" json:"changeFailureWindowMinutes,omitempty"`
}

func (ChronosphereScopeConfig) TableName() string {
	return "_tool_chronosphere_scope_configs"
}

// ChronosphereApiParams is used as the key for raw data tables.
type ChronosphereApiParams struct {
	ConnectionId uint64
	Namespace    string
}

// Compile-time check: ChronosphereAlertEvent → devops domain (for future converter).
var _ = (*devops.CicdDeploymentCommit)(nil)
