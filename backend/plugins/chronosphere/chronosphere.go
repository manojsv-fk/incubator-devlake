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

// Package main is the entry point for the chronosphere DevLake plugin.
// Chronosphere is FourKites' observability platform; we collect alert events
// to compute MTTR and Change Failure Rate as part of the DORA metrics suite.
package main

import (
	"github.com/apache/incubator-devlake/core/runner"
	"github.com/apache/incubator-devlake/plugins/chronosphere/impl"
	"github.com/spf13/cobra"
)

var PluginEntry impl.Chronosphere

func main() {
	cmd := &cobra.Command{Use: "chronosphere"}
	connectionId := cmd.Flags().Uint64P("connectionId", "c", 0, "connection id")
	namespace := cmd.Flags().StringP("namespace", "n", "", "Chronosphere namespace to collect")
	timeAfter := cmd.Flags().StringP("timeAfter", "a", "", "collect data after this time (RFC3339)")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		runner.DirectRun(cmd, args, PluginEntry, map[string]interface{}{
			"connectionId": *connectionId,
			"namespace":    *namespace,
		}, *timeAfter)
	}
	runner.RunCmd(cmd)
}
