// Copyright (c) 2024 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License.AGPL.txt in the project root for license information.

package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gitpod-io/gitpod/common-go/log"
	api "github.com/gitpod-io/gitpod/ide-metrics-api"
)

const (
	BackendPluginIncompatibleMetric = "supervisor_jb_backend_plugin_incompatible_total"
	BackendPluginStatusMetric       = "supervisor_jb_backend_plugin_status_total"
)

// update with go build
// go build -trimpath -ldflags "-buildid= -w -s -X 'github.com/gitpod-io/gitpod/jetbrains/launcher/pkg/metrics.SendRequests=true'"
var SendRequests = "false"

var gitpodHost = strings.Replace(os.Getenv("GITPOD_HOST"), "https://", "", -1)

func AddBackendPluginIncompatibleTotal(ide string) {
	if gitpodHost == "" {
		log.Error("no GITPOD_HOST env")
		return
	}
	doAddCounter(gitpodHost, BackendPluginIncompatibleMetric, map[string]string{"ide": ide}, 1)
}

type PluginStatus string

const (
	PluginStatusLoaded  = "loaded"
	PluginStatusStarted = "started"
)

func AddBackendPluginStatus(ide string, status PluginStatus) {
	if gitpodHost == "" {
		log.Error("no GITPOD_HOST env")
		return
	}
	doAddCounter(gitpodHost, BackendPluginStatusMetric, map[string]string{"ide": ide, "status": string(status)}, 1)
}

func doAddCounter(gitpodHost string, name string, labels map[string]string, value uint64) {
	if SendRequests == "false" {
		log.Info("do not send ide-metrics requests")
		return
	}
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
