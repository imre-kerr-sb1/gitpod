// Copyright (c) 2024 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License.AGPL.txt in the project root for license information.

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

const (
	TestInCompatibleString = "The Gitpod Remote (id=io.gitpod.jetbrains.remote, path=/workspace/.config/JetBrains/RemoteDev-PS/plugins/gitpod-remote, version=0.0.1-stable) plugin Plugin 'Gitpod Remote' (version '0.0.1-stable') is not compatible with the current version of the IDE, because it requires build 241.15989 or newer but the current build is PS-241.14494.237"
	TestGitpodGatewayLink  = "Gitpod gateway link"
)

func TestLauncherLogAnalyzer_Analyze(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {

		cmd := exec.Command("bash", "-c", fmt.Sprintf(`
sleep 0.2
echo happy testing
echo "%s"
sleep 0.1
`, TestGitpodGatewayLink))
		l := NewLauncherLogAnalyzer(nil, cmd)
		ctx, cancel := context.WithCancel(context.Background())
		if err := l.Analyze(ctx); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if err := cmd.Start(); err != nil {
			t.Errorf("unexpected start error: %v", err)
		}
		if err := cmd.Wait(); err != nil {
			t.Errorf("unexpected wait error: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
		cancel()
		if l.isBackendPluginStarted != true {
			t.Errorf("expected isBackendPluginStarted to be true, but got false")
		}
	})
}

func TestIdeaLogFileAnalyzer_Analyze(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		logPath := t.TempDir() + "/TestIdeaLogFileAnalyzer_Analyze.log"

		appendLogs := func(writeCompatibleStr bool) {
			tmpFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			defer tmpFile.Close()
			if _, err = tmpFile.WriteString("happy testing\n"); err != nil {
				t.Errorf("unexpected write error: %v", err)
			}
			if writeCompatibleStr {
				if _, err = tmpFile.WriteString(TestInCompatibleString + "\n"); err != nil {
					t.Errorf("unexpected write error: %v", err)
				}
			}
		}

		ctx, cancel := context.WithCancel(context.Background())
		l := NewIdeaLogAnalyzer(nil, logPath)
		if err := l.Analyze(ctx); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
		_, _ = os.Create(logPath)
		defer os.Remove(logPath)

		appendLogs(false)
		time.Sleep(20 * time.Millisecond)
		appendLogs(false)
		time.Sleep(20 * time.Millisecond)
		appendLogs(true)
		appendLogs(true)
		time.Sleep(300 * time.Millisecond)
		cancel()
		if l.inCompatibleBackendPlugin != true {
			t.Errorf("expected inCompatibleBackendPlugin to be true, but got false")
		}
	})
}
