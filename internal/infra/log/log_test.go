// Copyright The MatrixHub Authors.
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

package log

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type testPayload struct{}

func (testPayload) LoggerOutput(simple bool) []any {
	return []any{"simple", simple}
}

func TestLoggerConfigurationAndNonFatalMethods(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "matrixhub.log")
	require.NoError(t, SetLoggerWithConfig(true, Config{FilePath: &logPath, FileMaxSize: 1, FileMaxBackups: 1, FileMaxAges: 1}))
	t.Cleanup(func() { require.NoError(t, SetLoggerWithConfig(false, Config{})) })

	Debug("debug")
	Debugw("debugw", "key", "value")
	Debugf("debugf %d", 1)
	DebugWithPayload("debug payload", testPayload{})
	Info("info")
	Infow("infow", "key", "value")
	Infof("infof %d", 1)
	InfoWithPayload("info payload", testPayload{}, "key", "value")
	Warn("warn")
	Warnw("warnw", "key", "value")
	Warnf("warnf %d", 1)
	WarnWithPayload("warn payload", testPayload{}, "key", "value")
	Error("error")
	Errorw("errorw", "key", "value")
	Errorf("errorf %d", 1)
	ErrorWithPayload("error payload", testPayload{}, "key", "value")

	named := NewLogger("domain")
	require.NotNil(t, named.Logger("child"))
	require.NotNil(t, named.LoggerWithLevel("warn-only", zapcore.WarnLevel))
	require.NotNil(t, named.LoggerWithOptions("caller", zap.AddCallerSkip(1)))
	named.Debug("debug")
	named.Debugw("debugw", "key", "value")
	named.Debugf("debugf %d", 1)
	named.DebugWithPayload("debug payload", testPayload{}, "key", "value")
	named.Info("info")
	named.Infow("infow", "key", "value")
	named.Infof("infof %d", 1)
	named.InfoWithPayload("info payload", testPayload{}, "key", "value")
	named.Warn("warn")
	named.Warnw("warnw", "key", "value")
	named.Warnf("warnf %d", 1)
	named.WarnWithPayload("warn payload", testPayload{}, "key", "value")
	named.Error("error")
	named.Errorw("errorw", "key", "value")
	named.Errorf("errorf %d", 1)
	named.ErrorWithPayload("error payload", testPayload{}, "key", "value")

	Sync()
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	require.Contains(t, string(content), "info payload")
	require.Contains(t, string(content), "error payload")
}

func TestLoggerPanicMethods(t *testing.T) {
	require.NoError(t, SetLoggerWithConfig(false, Config{}))
	named := NewLogger("panic-test")
	require.Panics(t, func() { named.Panic("panic") })
	require.Panics(t, func() { named.Panicf("panic %d", 1) })
	require.Panics(t, func() { named.Panicw("panic", "key", "value") })
	require.Panics(t, func() { Panic("panic") })
	require.Panics(t, func() { Panicf("panic %d", 1) })
	require.Panics(t, func() { Panicw("panic", "key", "value") })
}

func TestLoggerProductionConfiguration(t *testing.T) {
	require.NoError(t, SetLoggerWithConfig(false, Config{Level: "warn", Encoder: "json"}))
	require.NoError(t, SetLoggerWithConfig(false, Config{Level: "not-a-level"}))
}
