// Copyright (c) 2024 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License.AGPL.txt in the project root for license information.

package main

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestIdeaLogFileAnalyzer_Analyze(t *testing.T) {

	prepare := func(t *testing.T, logName, testStr string, shouldWriteTestStr bool) (*IdeaLogFileAnalyzer, func()) {
		logPath := t.TempDir() + "/" + logName
		appendLogs := func(shouldWriteTestStr bool) {
			tmpFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			defer tmpFile.Close()
			if _, err = tmpFile.WriteString("happy testing\n"); err != nil {
				t.Errorf("unexpected write error: %v", err)
			}
			if shouldWriteTestStr {
				if _, err = tmpFile.WriteString(testStr + "\n"); err != nil {
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
		if shouldWriteTestStr {
			appendLogs(true)
			appendLogs(true)
		} else {
			appendLogs(false)
			appendLogs(false)
		}
		time.Sleep(300 * time.Millisecond)
		return l, cancel
	}

	type args struct {
		logfile            string
		ouputTestStr       string
		shouldWriteTestStr bool
		ruleName           string
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "should record plugin started",
			args: args{
				logfile:            "plugin_started.log",
				ouputTestStr:       "Gitpod gateway link",
				shouldWriteTestStr: true,
				ruleName:           "plugin started",
			},
		},
		{
			name: "should record plugin loaded",
			args: args{
				logfile:            "plugin_loaded.log",
				ouputTestStr:       "Loaded custom plugins: Gitpod Remote (0.0.1-stable)",
				shouldWriteTestStr: true,
				ruleName:           "plugin loaded",
			},
		},
		{
			name: "should record plugin incompatible",
			args: args{
				logfile:            "plugin_incompatible.log",
				ouputTestStr:       "The Gitpod Remote (id=io.gitpod.jetbrains.remote, path=/workspace/.config/JetBrains/RemoteDev-PS/plugins/gitpod-remote, version=0.0.1-stable) plugin Plugin 'Gitpod Remote' (version '0.0.1-stable') is not compatible with the current version of the IDE, because it requires build 241.15989 or newer but the current build is PS-241.14494.237",
				shouldWriteTestStr: true,
				ruleName:           "plugin incompatible",
			},
		},
		{
			name: "should record nothing",
			args: args{
				logfile:            "plugin_started.log",
				ouputTestStr:       "Gitpod gateway link",
				shouldWriteTestStr: false,
				ruleName:           "plugin started",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, cancel := prepare(t, tt.args.logfile, tt.args.ouputTestStr, tt.args.shouldWriteTestStr)
			defer cancel()
			shouldBeTrue := false
			for _, rule := range l.rules {
				if rule.Name == tt.args.ruleName {
					shouldBeTrue = true
					if !tt.args.shouldWriteTestStr {
						if rule.matched {
							t.Errorf("expected rule to not matched")
						}
						continue
					}
					if !rule.matched {
						t.Errorf("expected rule to be matched")
					}
				} else {
					if rule.matched {
						t.Errorf("expected other rules to not matched")
					}
				}
			}
			if !shouldBeTrue {
				t.Errorf("unexpected found not rule")
			}
		})
	}
}
