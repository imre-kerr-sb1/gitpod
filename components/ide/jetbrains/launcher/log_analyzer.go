// Copyright (c) 2024 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License.AGPL.txt in the project root for license information.

package main

import (
	"context"
	"os"
	"path"
	"regexp"
	"sync"

	"github.com/gitpod-io/gitpod/common-go/log"
	"github.com/gitpod-io/gitpod/jetbrains/launcher/pkg/metrics"
	"github.com/nxadm/tail"
)

type LogAnalyzer interface {
	Analyze(ctx context.Context) error
}

const (
	logDir = "/tmp/ide-desktop-log"
)

func init() {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.WithError(err).Error("failed to create logDir")
	}
}

type LineMatchRule struct {
	Name           string         // rule name
	Pattern        *regexp.Regexp // regex to match line
	LogFile        string         // print matched line to log file
	matchedHandler func()

	matched bool
}

// IdeaLogFileAnalyzer watches the idea.log file and does diagnostic on the output
type IdeaLogFileAnalyzer struct {
	launchCtx *LaunchContext
	path      string
	rules     []*LineMatchRule
	wg        sync.WaitGroup
}

var _ LogAnalyzer = &IdeaLogFileAnalyzer{}

func NewIdeaLogAnalyzer(launchCtx *LaunchContext, logPath string) *IdeaLogFileAnalyzer {
	ide := "unknown"
	if launchCtx != nil {
		ide = launchCtx.alias
	}
	logDir := path.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// if error and the directory does not exist, watcher will close chan directly
		// so no need to return error here
		log.WithError(err).Error("failed to create log file's directory")
	}

	l := &IdeaLogFileAnalyzer{
		path:      logPath,
		launchCtx: launchCtx,
		rules: []*LineMatchRule{
			{Name: "plugin started", Pattern: regexp.MustCompile(`Gitpod gateway link`), LogFile: "jb-backend-started.log", matchedHandler: func() {
				metrics.AddBackendPluginStatus(ide, metrics.PluginStatusStarted)
			}},
			{Name: "plugin loaded", Pattern: regexp.MustCompile(`Loaded custom plugins: Gitpod Remote`), LogFile: "jb-backend-loaded.log", matchedHandler: func() {
				metrics.AddBackendPluginStatus(ide, metrics.PluginStatusLoaded)
			}},
			{Name: "plugin incompatible", Pattern: regexp.MustCompile(`Plugin 'Gitpod Remote' .* is not compatible`), LogFile: "jb-backend-incompatible.log", matchedHandler: func() {
				metrics.AddBackendPluginIncompatibleTotal(ide)
			}},
		},
	}
	return l
}

func (l *IdeaLogFileAnalyzer) Wait() {
	l.wg.Wait()
}

func (l *IdeaLogFileAnalyzer) Analyze(ctx context.Context) error {
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		t, err := tail.TailFile(l.path, tail.Config{
			Follow:    true,
			ReOpen:    true,
			MustExist: false,
		})
		if err != nil {
			log.WithError(err).Error("failed to tail file")
			return
		}
		defer func() {
			_ = t.Stop()
		}()

		for {
			select {
			case line, ok := <-t.Lines:
				if !ok {
					log.Info("watcher chan closed")
					return
				}
				if line.Err != nil {
					log.WithError(line.Err).Warn("error reading line")
					continue
				}
				for _, rule := range l.rules {
					if rule.matched || !rule.Pattern.Match([]byte(line.Text)) {
						continue
					}
					rule.matched = true
					log.WithField("line", line.Text).WithField("rule", rule.Name).Info("matched rule")
					writeToFile(rule.LogFile, line.Text)
					rule.matchedHandler()
				}
			case <-ctx.Done():
				log.Info("stopping file watcher")
				return
			}
		}
	}()
	return nil
}

func writeToFile(fileName string, line string) {
	f, err := os.OpenFile(logDir+"/"+fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.WithError(err).Error("failed to open file")
		return
	}
	defer f.Close()
	if _, err := f.WriteString(line); err != nil {
		log.WithError(err).Error("failed to write to file")
	}
}
