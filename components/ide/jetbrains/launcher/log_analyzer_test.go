// Copyright (c) 2024 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License.AGPL.txt in the project root for license information.

package main

import (
	"os/exec"
	"testing"
)

func TestLauncherLogDiagnostic_Diagnostic(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		l := &LauncherLogAnalyzer{}
		cmd := exec.Command("echo", "The Gitpod Remote (id=io.gitpod.jetbrains.remote, path=/workspace/.config/JetBrains/RemoteDev-PS/plugins/gitpod-remote, version=0.0.1-stable) plugin Plugin 'Gitpod Remote' (version '0.0.1-stable') is not compatible with the current version of the IDE, because it requires build 241.15989 or newer but the current build is PS-241.14494.237")
		if err := l.Analyze(cmd, "test.log"); err != nil {
			t.Errorf("unexpected diagnostic error: %v", err)
		}
		if err := cmd.Start(); err != nil {
			t.Errorf("unexpected start error: %v", err)
		}
		if err := cmd.Wait(); err != nil {
			t.Errorf("unexpected wait error: %v", err)
		}
		l.Stop()
		if l.inCompatibleBackendPlugin != true {
			t.Errorf("expected inCompatibleBackendPlugin to be true, but got false")
		}
	})
}
