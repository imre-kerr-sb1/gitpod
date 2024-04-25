// Copyright (c) 2024 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License.AGPL.txt in the project root for license information.

package main

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sync"

	"github.com/gitpod-io/gitpod/common-go/log"
	"github.com/nxadm/tail"
)

type LogAnalyzer interface {
	Analyze(ctx context.Context) error
	Wait()
}

const (
	logDir                       = "/var/log/gitpod"
	backendStartedPatternLogFile = "jb-backend-started.log"
)

var (
	inCompatiblePattern   = regexp.MustCompile(`Plugin 'Gitpod Remote' .* is not compatible`)
	backendStartedPattern = regexp.MustCompile(`Gitpod gateway link`)
)

// IdeaLogFileAnalyzer watches the idea.log file and does diagnostic on the output
type IdeaLogFileAnalyzer struct {
	path                      string
	inCompatibleBackendPlugin bool
	wg                        sync.WaitGroup
	launchCtx                 *LaunchContext
}

var _ LogAnalyzer = &IdeaLogFileAnalyzer{}

func NewIdeaLogAnalyzer(launchCtx *LaunchContext, path string) *IdeaLogFileAnalyzer {
	return &IdeaLogFileAnalyzer{
		path:      path,
		launchCtx: launchCtx,
	}
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

		for {
			select {
			case line := <-t.Lines:
				if line.Err != nil {
					log.WithError(line.Err).Warn("error reading line")
					continue
				}
				if !l.inCompatibleBackendPlugin && inCompatiblePattern.Match([]byte(line.Text)) {
					l.inCompatibleBackendPlugin = true
					AddBackendPluginIncompatibleTotal(getIdeName(l.launchCtx))
					log.WithField("line", line).Error("backend plugin is not compatible")
				}
			case <-ctx.Done():
				log.Info("Stopping file watcher...")
				_ = t.Stop()
				return
			}
		}
	}()
	return nil
}

// LauncherLogAnalyzer pipes JetBrains remote-dev-server.sh 's stdout and
// stderr into os.Stdout and os.Stderr, and does diagnostic on the output
type LauncherLogAnalyzer struct {
	reader                 *io.PipeReader
	writer                 *io.PipeWriter
	isBackendPluginStarted bool
	cmd                    *exec.Cmd
	wg                     sync.WaitGroup
	launchCtx              *LaunchContext
}

func NewLauncherLogAnalyzer(launchCtx *LaunchContext, cmd *exec.Cmd) *LauncherLogAnalyzer {
	return &LauncherLogAnalyzer{
		cmd:       cmd,
		launchCtx: launchCtx,
	}
}

var _ LogAnalyzer = &LauncherLogAnalyzer{}

func (l *LauncherLogAnalyzer) Analyze(ctx context.Context) error {
	l.wg.Add(1)
	reader, writer := io.Pipe()
	l.reader = reader
	l.writer = writer
	stdout := io.MultiWriter(os.Stdout, writer)
	stderr := io.MultiWriter(os.Stderr, writer)
	l.cmd.Stdout = stdout
	l.cmd.Stderr = stderr
	go func() {
		<-ctx.Done()
		writer.Close()
	}()
	go l.doAnalyze()
	return nil
}

func (l *LauncherLogAnalyzer) Wait() {
	l.wg.Wait()
}

func (l *LauncherLogAnalyzer) doAnalyze() {
	defer l.wg.Done()
	scanner := bufio.NewScanner(l.reader)
	for scanner.Scan() {
		line := scanner.Text()
		if !l.isBackendPluginStarted && backendStartedPattern.Match([]byte(line)) {
			l.isBackendPluginStarted = true
			AddBackendPluginStartedTotal(getIdeName(l.launchCtx))
			log.WithField("line", line).Error("backend plugin is not compatible")
			l.WriteToFile(backendStartedPatternLogFile, line)
		}
	}
}

func (l *LauncherLogAnalyzer) WriteToFile(fileName string, line string) {
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

func getIdeName(ctx *LaunchContext) string {
	if ctx == nil {
		return "unknown"
	}
	return ctx.alias
}
