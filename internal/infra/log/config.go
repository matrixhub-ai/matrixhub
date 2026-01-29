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

type Config struct {
	Level   string `yaml:"level"`
	Encoder string `yaml:"encoder"`

	FilePath       *string `yaml:"filePath"`       // 旧的日志文件在相同目录下
	FileMaxSize    int     `yaml:"fileMaxSize"`    // 单个日志文件的最大大小(MB)
	FileMaxBackups int     `yaml:"fileMaxBackups"` // 保留旧日志文件的最大数量
	FileMaxAges    int     `yaml:"fileMaxAges"`    // 保留旧文件的最大天数, 使用文件名和时间戳判断
	Compress       bool    `yaml:"compress"`       // 是否压缩归档旧文件
}
