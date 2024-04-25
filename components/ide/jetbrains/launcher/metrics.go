// Copyright (c) 2024 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License.AGPL.txt in the project root for license information.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/gitpod-io/gitpod/common-go/log"
	api "github.com/gitpod-io/gitpod/ide-metrics-api"
)

const (
	BackendPluginIncompatibleName = "supervisor_jb_backend_plugin_incompatible_total"
)

func AddBackendPluginIncompatibleTotal(ide string) {
	host := os.Getenv("GITPOD_HOST")
	if host == "" {
		log.Error("no GITPOD_HOST env")
		return
	}
	doAddCounter(host, BackendPluginIncompatibleName, map[string]string{"ide": ide}, 1)
}

func AddBackendPluginStartedTotal(ide string) {
	host := os.Getenv("GITPOD_HOST")
	if host == "" {
		log.Error("no GITPOD_HOST env")
		return
	}
	// TODO:
}

func doAddCounter(gitpodHost string, name string, labels map[string]string, value uint64) {
	req := &api.AddCounterRequest{
		Name:   name,
		Labels: labels,
		Value:  int32(value),
	}
	log.WithField("req", req).Debug("jetbrains-launcher: gprc metric: add counter")

	body, err := json.Marshal(req)
	if err != nil {
		log.WithField("req", req).WithError(err).Error("jetbrains-launcher: grpc metric: failed to marshal request")
		return
	}
	url := fmt.Sprintf("https://ide.%s/metrics-api/metrics/counter/add/%s", gitpodHost, name)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.WithError(err).Error("jetbrains-launcher: grpc metric: failed to create request")
		return
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Client", "jetbrains-launcher")
	resp, err := http.DefaultClient.Do(request)
	var statusCode int
	if resp != nil {
		statusCode = resp.StatusCode
	}
	if err == nil && statusCode == http.StatusOK {
		return
	}
	var respBody string
	var status string
	if resp != nil {
		status = resp.Status
		body, _ := ioutil.ReadAll(resp.Body)
		if body != nil {
			respBody = string(body)
		}
	}
	log.WithField("url", url).
		WithField("req", req).
		WithField("statusCode", statusCode).
		WithField("status", status).
		WithField("respBody", respBody).
		WithError(err).
		Error("jetbrains-launcher: grpc metric: failed to add counter")
}
