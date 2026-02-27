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
	"context"
	"net/http"

	"github.com/apache/incubator-devlake/server/api/shared"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/aicode/models"
)

// AiCodeTestConnResponse is returned by the connection test endpoint.
type AiCodeTestConnResponse struct {
	shared.ApiBody
	Login string `json:"login"`
}

// testConnection validates credentials by calling GET /user on the GitHub API.
func testConnection(ctx context.Context, conn models.AiCodeConn) (*AiCodeTestConnResponse, errors.Error) {
	if vld != nil {
		if err := vld.Struct(conn); err != nil {
			return nil, errors.Default.Wrap(err, "invalid connection parameters")
		}
	}

	apiClient, err := helper.NewApiClientFromConnection(ctx, basicRes, &conn)
	if err != nil {
		return nil, err
	}
	apiClient.SetHeaders(map[string]string{
		"Accept":               "application/vnd.github+json",
		"X-GitHub-Api-Version": "2022-11-28",
	})

	res, err := apiClient.Get("user", nil, nil)
	if err != nil {
		return nil, err
	}
	if res.StatusCode == http.StatusUnauthorized {
		return nil, errors.HttpStatus(http.StatusUnauthorized).New("invalid GitHub token – check PAT scopes")
	}
	if res.StatusCode != http.StatusOK {
		return nil, errors.HttpStatus(res.StatusCode).New("unexpected status from GitHub API")
	}

	// Decode the login name from the response.
	var userBody struct {
		Login string `json:"login"`
	}
	if err := helper.UnmarshalResponse(res, &userBody); err != nil {
		return nil, err
	}

	body := &AiCodeTestConnResponse{}
	body.Success = true
	body.Message = "connection successful"
	body.Login = userBody.Login
	return body, nil
}

// TestConnection tests the provided credentials without saving.
// @Summary  Test a new aicode connection
// @Tags     plugins/aicode
// @Param    body body models.AiCodeConn true "connection credentials"
// @Success  200 {object} AiCodeTestConnResponse
// @Router   /plugins/aicode/test [post]
func TestConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	var conn models.AiCodeConn
	if err := helper.Decode(input.Body, &conn, vld); err != nil {
		return nil, err
	}
	result, err := testConnection(context.TODO(), conn)
	if err != nil {
		return nil, err
	}
	return &plugin.ApiResourceOutput{Body: result, Status: http.StatusOK}, nil
}

// TestExistingConnection tests an already-saved connection by its ID.
func TestExistingConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	connection, err := dsHelper.ConnApi.GetMergedConnection(input)
	if err != nil {
		return nil, errors.BadInput.Wrap(err, "find connection from db")
	}
	result, err := testConnection(context.TODO(), connection.AiCodeConn)
	if err != nil {
		return nil, err
	}
	return &plugin.ApiResourceOutput{Body: result, Status: http.StatusOK}, nil
}

// PostConnections creates a new connection.
func PostConnections(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ConnApi.Post(input)
}

// ListConnections lists all saved connections.
func ListConnections(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ConnApi.List(input)
}

// GetConnection fetches a single connection by ID.
func GetConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ConnApi.Get(input)
}

// PatchConnection updates fields on an existing connection.
func PatchConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ConnApi.Patch(input)
}

// DeleteConnection removes a connection.
func DeleteConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ConnApi.Delete(input)
}
