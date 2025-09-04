// Copyright 2025.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"log/slog"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

func ConfigureLogger(logFile string, debug bool) {
	writer := &lumberjack.Logger{
		Filename:   logFile,
		LocalTime:  true,
		MaxBackups: 10,
		MaxSize:    10,
	}
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				a.Value = slog.AnyValue(&slog.Source{
					File: filepath.Base(a.Value.Any().(*slog.Source).File),
					Line: a.Value.Any().(*slog.Source).Line,
				})
			}
			return a
		},
	})
	slog.SetDefault(slog.New(handler))
}
