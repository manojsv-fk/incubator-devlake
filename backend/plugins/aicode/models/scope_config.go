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
)

// AiCodeScopeConfig configures how AI attribution is detected and categorised
// for a given scope (org). All fields accept Go regex patterns.
type AiCodeScopeConfig struct {
	common.ScopeConfig `mapstructure:",squash" json:",inline" gorm:"embedded"`

	// CopilotTrailerPattern matches commit-message trailers that indicate
	// GitHub Copilot authorship. Default covers the standard trailer.
	// Example: "(?i)co-authored-by:.*copilot"
	CopilotTrailerPattern string `gorm:"type:varchar(512)" mapstructure:"copilotTrailerPattern,omitempty" json:"copilotTrailerPattern,omitempty"`

	// ClaudeTrailerPattern matches commit-message trailers for Claude Code.
	// Example: "(?i)co-authored-by:.*claude"
	ClaudeTrailerPattern string `gorm:"type:varchar(512)" mapstructure:"claudeTrailerPattern,omitempty" json:"claudeTrailerPattern,omitempty"`

	// GenericAiTrailerPattern is a catch-all for any other AI tool trailers.
	// Example: "(?i)(generated-by|co-authored-by):.*ai"
	GenericAiTrailerPattern string `gorm:"type:varchar(512)" mapstructure:"genericAiTrailerPattern,omitempty" json:"genericAiTrailerPattern,omitempty"`

	// AiSurvivalDays is the number of days after which we check whether AI-generated
	// lines are still present in the file (for "AI Code Survival Rate" metric).
	// Defaults to 14.
	AiSurvivalDays int `gorm:"default:14" mapstructure:"aiSurvivalDays,omitempty" json:"aiSurvivalDays,omitempty"`
}

func (AiCodeScopeConfig) TableName() string {
	return "_tool_aicode_scope_configs"
}

func (sc *AiCodeScopeConfig) SetConnectionId(target *AiCodeScopeConfig, connectionId uint64) {
	target.ConnectionId = connectionId
	target.ScopeConfig.ConnectionId = connectionId
}

// DefaultScopeConfig returns sensible defaults so teams can get started quickly.
func DefaultScopeConfig() AiCodeScopeConfig {
	return AiCodeScopeConfig{
		CopilotTrailerPattern:   `(?i)co-authored-by:.*copilot`,
		ClaudeTrailerPattern:    `(?i)co-authored-by:.*claude`,
		GenericAiTrailerPattern: `(?i)(generated-by|co-authored-by):.*\b(ai|bot)\b`,
		AiSurvivalDays:          14,
	}
}
