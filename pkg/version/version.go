/*
Copyright The MatrixHub Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package version

// Base version information.
//
// This is the fallback data used when version information from git is not
// provided via go ldflags.
//
// If you are looking at these fields in the git tree, they look
// strange. They are modified on the fly by the build process.
import (
	"fmt"
	"runtime"
)

var (
	GitVersion = "v0.0.0-main"
	GitCommit  = "abcd01234"            // sha1 from git, output of $(git rev-parse HEAD)
	BuildDate  = "1970-01-01T00:00:00Z" // RFC3339 fallback when ldflags are absent
)

type Info struct {
	GitVersion string
	GitCommit  string
	GoVersion  string
	BuildDate  string
	Compiler   string
	Platform   string
}

func Get() Info {
	return Info{
		GitVersion: GitVersion,
		GitCommit:  GitCommit,
		GoVersion:  runtime.Version(),
		BuildDate:  BuildDate,
		Compiler:   runtime.Compiler,
		Platform:   fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

func PrintVersion() {
	info := Get()
	fmt.Printf("Version:      %s\n", info.GitVersion)
	fmt.Printf("Git Commit:   %s\n", info.GitCommit)
	fmt.Printf("Go Version:   %s\n", info.GoVersion)
	fmt.Printf("Build Date:   %s\n", info.BuildDate)
	fmt.Printf("Compiler:     %s\n", info.Compiler)
	fmt.Printf("Platform:     %s\n", info.Platform)
}
