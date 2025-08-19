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
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New constructs a Sugared Logger that writes to a file and
// provides human-readable timestamps.
func New(service string, outputPaths ...string) (*zap.SugaredLogger, error) {
	config := zap.NewProductionConfig()

	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)

	config.DisableStacktrace = true
	config.InitialFields = map[string]any{
		"service": service,
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	logDir := filepath.Join(home, ".lazyssh")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, err
	}
	config.OutputPaths = []string{filepath.Join(logDir, "lazyssh.log")}

	if outputPaths != nil {
		config.OutputPaths = outputPaths
	}

	log, err := config.Build(zap.WithCaller(true))
	if err != nil {
		return nil, err
	}

	return log.Sugar(), nil
}
