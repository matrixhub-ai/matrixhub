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
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	sugar *zap.SugaredLogger
	name  string
}

type LoggerPayload interface {
	LoggerOutput(simple bool) (keysAndValues []any)
}

func NewLogger(name string) *Logger {
	loggerLock.Lock()
	defer loggerLock.Unlock()

	return logger.LoggerWithOptions(name, zap.AddCallerSkip(-1))
}

func (logger *Logger) Logger(name string) *Logger {
	return &Logger{
		name:  name,
		sugar: logger.sugar.Named(name),
	}
}

func (logger *Logger) LoggerWithLevel(name string, level zapcore.LevelEnabler) *Logger {
	return logger.LoggerWithOptions(name, zap.IncreaseLevel(level))
}

func (logger *Logger) LoggerWithOptions(name string, opts ...zap.Option) *Logger {
	return &Logger{
		name:  name,
		sugar: logger.sugar.Desugar().WithOptions(opts...).Named(name).Sugar(),
	}
}

func (logger *Logger) InfoWithPayload(msg string, payload LoggerPayload, keysAndValues ...any) {
	keysAndValues = append(keysAndValues, payload.LoggerOutput(true)...)
	logger.sugar.Infow(msg, keysAndValues...)
}

func (logger *Logger) WarnWithPayload(msg string, payload LoggerPayload, keysAndValues ...any) {
	keysAndValues = append(keysAndValues, payload.LoggerOutput(true)...)
	logger.sugar.Warnw(msg, keysAndValues...)
}

func (logger *Logger) ErrorWithPayload(msg string, payload LoggerPayload, keysAndValues ...any) {
	keysAndValues = append(keysAndValues, payload.LoggerOutput(true)...)
	logger.sugar.Errorw(msg, keysAndValues...)
}

func (logger *Logger) DebugWithPayload(msg string, payload LoggerPayload, keysAndValues ...any) {
	keysAndValues = append(keysAndValues, payload.LoggerOutput(false)...)
	logger.sugar.Debugw(msg, keysAndValues...)
}

func (logger *Logger) Debug(args ...any) {
	logger.sugar.Debug(args...)
}

func (logger *Logger) Debugw(msg string, keysAndValues ...any) {
	logger.sugar.Debugw(msg, keysAndValues...)
}

func (logger *Logger) Debugf(template string, args ...any) {
	logger.sugar.Debugf(template, args...)
}

func (logger *Logger) Info(args ...any) {
	logger.sugar.Info(args...)
}

func (logger *Logger) Infow(msg string, keysAndValues ...any) {
	logger.sugar.Infow(msg, keysAndValues...)
}

func (logger *Logger) Infof(template string, args ...any) {
	logger.sugar.Infof(template, args...)
}

func (logger *Logger) Warn(args ...any) {
	logger.sugar.Warn(args...)
}

func (logger *Logger) Warnf(template string, args ...any) {
	logger.sugar.Warnf(template, args...)
}

func (logger *Logger) Warnw(msg string, keysAndValues ...any) {
	logger.sugar.Warnw(msg, keysAndValues...)
}

func (logger *Logger) Error(args ...any) {
	logger.sugar.Error(args)
}

func (logger *Logger) Errorf(template string, args ...any) {
	logger.sugar.Errorf(template, args...)
}

func (logger *Logger) Errorw(msg string, keysAndValues ...any) {
	logger.sugar.Errorw(msg, keysAndValues...)
}

func (logger *Logger) Fatal(args ...any) {
	logger.sugar.Fatal(args...)
}

func (logger *Logger) Fatalf(template string, args ...any) {
	logger.sugar.Fatalf(template, args...)
}

func (logger *Logger) Fatalw(msg string, keysAndValues ...any) {
	logger.sugar.Fatalw(msg, keysAndValues...)
}

func (logger *Logger) Panic(args ...any) {
	logger.sugar.Panic(args...)
}

func (logger *Logger) Panicf(template string, args ...any) {
	logger.sugar.Panicf(template, args...)
}

func (logger *Logger) Panicw(msg string, keysAndValues ...any) {
	logger.sugar.Panicw(msg, keysAndValues...)
}
