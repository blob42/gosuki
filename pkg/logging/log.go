//
// Copyright (c) 2023-2025 Chakib Ben Ziane <contact@blob42.xyz> and [`GoSuki` contributors]
// (https://github.com/blob42/gosuki/graphs/contributors).
//
// All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This file is part of GoSuki.
//
// GoSuki is free software: you can redistribute it and/or modify it under the terms of
// the GNU Affero General Public License as published by the Free Software Foundation,
// either version 3 of the License, or (at your option) any later version.
//
// GoSuki is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
// without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR
// PURPOSE.  See the GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License along with
// gosuki.  If not, see <http://www.gnu.org/licenses/>.

package logging

import (
	"fmt"
	"io"
	"maps"
	"math"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/blob42/gosuki/pkg/build"
	"github.com/charmbracelet/lipgloss"
	log "github.com/charmbracelet/log"
	"github.com/muesli/termenv"
)

const EnvGosukiDebug = "GOSUKI_DEBUG"

// log level shortcuts
const (
	Release = iota
	Dev
	Silent = math.MaxInt
)

// log level strings
var (
	traceLvl  string = "trace"
	debugLvl  string = "debug"
	infoLvl   string = "info"
	warnLvl   string = "warn"
	errLvl    string = "error"
	fatalLvl  string = "fatal"
	silentLvl string = "none"
	allLevels        = []string{
		traceLvl,
		debugLvl,
		infoLvl,
		warnLvl,
		errLvl,
		fatalLvl,
		silentLvl,
	}

	Stdout io.Writer
)

const (
	debugDefaultFmt = `%{color} %{time:15:04:05.000} %{level:.4s} %{shortfunc:.10s}: %{color:reset} %{message}`
	debugFmt        = `%{color} %{time:15:04:05.000} %{level:.4s} [%{module:.4s}] %{shortfile}:%{shortfunc:.10s}: %{color:reset} %{message}`
	releaseFmt      = `%{color}[%{level:.4s}]%{color:reset} %{message}`
)

var (
	//RELEASE: Change to Release for release mode
	LoggingMode = Dev
	TUIMode     bool
	SilentMode  bool
)

var (
	loggers = make(map[string]*Logger)

	globalLevel log.Level

	// logger lvl for each subsystem
	loggerLevels = make(map[string]log.Level)

	DefaultLogLevels = map[int]log.Level{
		Release: log.WarnLevel,
		Dev:     log.DebugLevel,
	}

	levels = map[string]log.Level{
		silentLvl: math.MaxInt,
		fatalLvl:  log.FatalLevel,
		errLvl:    log.ErrorLevel,
		warnLvl:   log.WarnLevel,
		infoLvl:   log.InfoLevel,
		debugLvl:  log.DebugLevel,
		traceLvl:  log.DebugLevel - 1,
	}

	logTextStyle = lipgloss.NewStyle().Foreground(
		lipgloss.AdaptiveColor{Light: "245", Dark: "252"},
	)
	logTextFaintStyle = lipgloss.NewStyle().Foreground(
		lipgloss.AdaptiveColor{Light: "240", Dark: "246"},
	)
	logLevelStyles = map[log.Level]lipgloss.Style{
		log.DebugLevel: lipgloss.NewStyle().
			SetString(strings.ToUpper(log.DebugLevel.String())).
			MaxWidth(4).
			Foreground(lipgloss.Color("63")),
		log.InfoLevel: lipgloss.NewStyle().
			SetString(strings.ToUpper(log.InfoLevel.String())).
			// Bold(true).
			MaxWidth(4).
			Foreground(lipgloss.Color("36")),
		log.WarnLevel: lipgloss.NewStyle().
			SetString(strings.ToUpper(log.WarnLevel.String())).
			MaxWidth(4).
			Foreground(lipgloss.Color("178")),
		log.ErrorLevel: lipgloss.NewStyle().
			SetString(strings.ToUpper(log.ErrorLevel.String())).
			MaxWidth(4).
			Foreground(lipgloss.Color("204")),
		log.FatalLevel: lipgloss.NewStyle().
			SetString(strings.ToUpper(log.FatalLevel.String())).
			MaxWidth(4).
			Foreground(lipgloss.Color("134")),
	}
)

type Logger struct {
	*log.Logger
}

func (l *Logger) With(keyvals ...any) *Logger {
	return &Logger{l.Logger.With(keyvals...)}
}

func (l *Logger) WithPrefix(prefix string) *Logger {
	return &Logger{l.Logger.WithPrefix(prefix)}
}

func NewLogger(w io.Writer) *Logger {
	l := new(Logger)
	logger := log.New(w)
	styles := log.DefaultStyles()
	styles.Prefix = lipgloss.NewStyle().Faint(true)
	styles.Levels[TraceLevel] = lipgloss.NewStyle().
		SetString("TRACE").
		Bold(true).
		MaxWidth(4).
		Foreground(lipgloss.Color("61"))
	logger.SetStyles(styles)
	l.Logger = logger
	return l
}

func isSilentMode() bool {
	rawArgs := strings.Join(os.Args, " ")
	if strings.Contains(rawArgs, "--silent") ||
		strings.Contains(rawArgs, "-S") {
		return true
	}
	return false
}

func GetLogger(module string) *Logger {
	module = strings.ToLower(module)
	lg := NewLogger(os.Stderr)

	if LoggingMode == Silent || SilentMode ||
		slices.Contains(build.Tags(), "silent") {
		return NewLogger(io.Discard)
	}

	lg.SetPrefix(fmt.Sprintf("%.4s", module))
	if LoggingMode == Dev {
		lg.SetTimeFormat(time.TimeOnly)
		lg.SetReportTimestamp(true)
		lg.SetCallerFormatter(func(file string, line int, _ string) string {
			return fmt.Sprintf("%s:%d", trimCallerPath(file, 1), line)
		})
		lg.SetReportCaller(true)
		lg.SetLevel(log.DebugLevel)

		//RELEASE:
	} else {
		if lvl, ok := loggerLevels[module]; ok {
			lg.SetLevel(lvl)
		} else {
			lg.SetLevel(globalLevel)
		}
	}

	loggers[strings.ToLower(strings.TrimSpace(module))] = lg

	return lg
}

func listLoggers() []string {
	return slices.DeleteFunc(
		slices.Collect(maps.Keys(loggers)),
		func(s string) bool { return s == "" },
	)
}

// func newReleaseLogger(module string) *log.Logger {
// 	if len(module) > 0 {
// 		return log.NewWithOptions(os.Stderr, log.Options{
// 			Prefix: fmt.Sprintf("[%.4s]", module),
// 		})
// 	} else {
// 		return log.New(os.Stderr)
// 	}
// }

// NewFileLogger creates a new logger that outputs to a specified file.
func NewFileLogger(fileName string) (*Logger, error) {
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	lg := NewLogger(file)
	lg.SetLevel(log.DebugLevel) // Set default level, can be adjusted as needed

	loggers[fileName] = lg
	return lg, nil
}

// flog is a convenience function for logging messages to a specified file logger.
func FDebugf(fileName, format string, args ...any) {
	var logger *Logger
	var err error

	logger, exists := loggers[fileName]
	if !exists {
		logger, err = NewFileLogger(fileName)
		if err != nil {
			panic(fmt.Sprintf("Failed to create logger: %v", err))
		}
		loggers[fileName] = logger
	}

	logger.Debugf(format, args...)
}

func SetLevel(lvl log.Level) {
	for _, logger := range loggers {
		// fmt.Printf("setting log level to:%v for %v\n ", lvl, logger)
		logger.SetLevel(lvl)

		// Silent mode
		if lvl == levels[silentLvl] {
			logger.SetOutput(io.Discard)
		} else {
			logger.SetOutput(os.Stderr)
		}
	}
}

func SetUnitLevel(u string, lvl log.Level) {
	if logger, ok := loggers[u]; ok {
		logger.SetLevel(lvl)
		// Silent mode
		if lvl == levels[silentLvl] {
			logger.SetOutput(io.Discard)
		} else {
			logger.SetOutput(os.Stderr)
		}
	}
}

// Sets the logging into TUI mode.
func SetTUI(output io.Writer) {
	Stdout = io.Discard
	TUIMode = true
	tuiLogStyles := log.DefaultStyles()
	tuiLogStyles.Levels = logLevelStyles

	tuiLogStyles.Message = logTextStyle
	tuiLogStyles.Value = logTextStyle

	tuiLogStyles.Prefix = logTextFaintStyle
	tuiLogStyles.Key = logTextFaintStyle
	tuiLogStyles.Separator = logTextFaintStyle

	for _, logger := range loggers {
		logger.SetOutput(output)
		// see https://github.com/charmbracelet/log?tab=readme-ov-file#styles
		logger.SetStyles(tuiLogStyles)
		logger.SetColorProfile(termenv.ANSI256)
		logger.SetReportCaller(false)
		logger.SetReportTimestamp(false)

		// effectively disables debug on tui
		if logger.GetLevel() < log.InfoLevel {
			logger.SetLevel(log.Level(log.InfoLevel))
		}

	}

}

func init() {
	Stdout = os.Stdout
	SilentMode = isSilentMode()
	envDebug := os.Getenv(EnvGosukiDebug)

	if envDebug != "" {
		if err := ParseDebugLevels(envDebug); err != nil {
			log.Fatal(err)
		}
	}
}
