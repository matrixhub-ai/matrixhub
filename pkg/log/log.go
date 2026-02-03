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
	"fmt"
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	loggerLock     sync.Mutex
	zapLogger      *zap.Logger
	zapSugarLogger *zap.SugaredLogger

	logger *Logger
)

func init() {
	debug := strings.ToLower(os.Getenv("DEBUG")) == "true"
	if err := SetLoggerWithConfig(debug, Config{}); err != nil {
		panic(fmt.Sprintf("log init failed: %s", err))
	}
}

func SetLoggerWithConfig(debug bool, config Config) error {
	loggerLock.Lock()
	defer loggerLock.Unlock()

	sink, err := openSinks(config)
	if err != nil {
		return err
	}

	var level zapcore.LevelEnabler
	var encoderConfig zapcore.EncoderConfig
	constructor := zapcore.NewConsoleEncoder
	if debug {
		level = zap.DebugLevel
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	} else {
		level = zap.InfoLevel
		if config.Level != "" {
			l := zap.NewAtomicLevel()
			if err = l.UnmarshalText([]byte(config.Level)); err == nil {
				level = l
			}
		}

		encoderConfig = zap.NewProductionEncoderConfig()

		if config.Encoder == "json" {
			constructor = zapcore.NewJSONEncoder
		} else {
			encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		}
	}
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	opts := []zap.Option{zap.AddCaller(), zap.AddCallerSkip(2), zap.AddStacktrace(zap.ErrorLevel)}
	if debug {
		opts = append(opts, zap.Development())
	}

	core := zapcore.NewCore(constructor(encoderConfig), sink, level)
	zapLogger = zap.New(core, opts...)
	zapSugarLogger = zapLogger.Sugar()

	logger = &Logger{sugar: zapSugarLogger}
	return nil
}

func Sync() {
	if err := zapLogger.Sync(); err != nil {
		logger.Warnw("sync zap log failed", "error", err.Error())
	}
}

func openSinks(config Config) (zapcore.WriteSyncer, error) {
	sink, _, err := zap.Open("stderr")
	if err != nil {
		return nil, err
	}

	if config.FilePath == "" {
		return sink, nil
	}

	lumberJackLogger := &lumberjack.Logger{
		Filename:   config.FilePath,
		MaxSize:    config.FileMaxSize,
		MaxBackups: config.FileMaxBackups,
		MaxAge:     config.FileMaxAges,
		Compress:   config.Compress,
	}
	return zap.CombineWriteSyncers(sink, zapcore.AddSync(lumberJackLogger)), nil
}

func Debugw(msg string, keysAndValues ...any) {
	logger.Debugw(msg, keysAndValues...)
}

func Infow(msg string, keysAndValues ...any) {
	logger.Infow(msg, keysAndValues...)
}

func Warnw(msg string, keysAndValues ...any) {
	logger.Warnw(msg, keysAndValues...)
}

func Errorw(msg string, keysAndValues ...any) {
	logger.Errorw(msg, keysAndValues...)
}
