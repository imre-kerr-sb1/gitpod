// Copyright (c) 2024 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License.AGPL.txt in the project root for license information.

package main

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"regexp"

	"github.com/gitpod-io/gitpod/common-go/log"
)

type LogAnalyzer interface {
	Analyze(cmd *exec.Cmd, logFileName string) error
	Stop()
}

const (
	logDir                     = "/var/log/gitpod"
	inCompatiblePatternLogFile = "jb-plugin-incompatible.log"
)

var (
	inCompatiblePattern = regexp.MustCompile(`Plugin 'Gitpod Remote' .* is not compatible`)
)

// LauncherLogAnalyzer pipes JetBrains remote-dev-server.sh 's stdout and
// stderr into os.Stdout and os.Stderr, and does diagnostic on the output
type LauncherLogAnalyzer struct {
	reader                    *io.PipeReader
	writer                    *io.PipeWriter
	inCompatibleBackendPlugin bool
}

var _ LogAnalyzer = &LauncherLogAnalyzer{}

func (l *LauncherLogAnalyzer) Analyze(cmd *exec.Cmd, logFileName string) error {
	reader, writer := io.Pipe()
	l.reader = reader
	l.writer = writer
	stdout := io.MultiWriter(os.Stdout, writer)
	stderr := io.MultiWriter(os.Stderr, writer)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	go l.doAnalyze()
	return nil
}

func (l *LauncherLogAnalyzer) Stop() {
	l.writer.Close()
}

func (l *LauncherLogAnalyzer) doAnalyze() {
	scanner := bufio.NewScanner(l.reader)
	for scanner.Scan() {
		line := scanner.Text()
		if !l.inCompatibleBackendPlugin && inCompatiblePattern.Match([]byte(line)) {
			l.inCompatibleBackendPlugin = true
			log.WithField("line", line).Error("backend plugin is not compatible")
			l.WriteToFile(inCompatiblePatternLogFile, line)
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
