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

package preset

import (
	"bytes"
	"context"
	"fmt"

	"github.com/spf13/viper"

	_ "embed"
)

type RegistrySpec struct {
	Name        string
	Description string
	URL         string
}

type ProjectSpec struct {
	Name string
	// Organization is the upstream namespace; empty means Name.
	Organization string
	Registry     string
}

func (s ProjectSpec) UpstreamOrganization() string {
	if s.Organization == "" {
		return s.Name
	}
	return s.Organization
}

type ModelSpec struct {
	Project string
	Name    string
	// Recommended lists the model among recommended models; nil means true.
	Recommended    *bool
	Size           int64
	ParameterCount int64
	Labels         []LabelSpec
}

func (s ModelSpec) IsRecommended() bool {
	return s.Recommended == nil || *s.Recommended
}

type LabelSpec struct {
	Category string
	Name     string
}

type Manifest struct {
	Registries []RegistrySpec
	Projects   []ProjectSpec
	Models     []ModelSpec
}

//go:embed default_manifest.yaml
var defaultManifestYAML []byte

// DefaultManifest returns an independent copy of the built-in manifest;
// upgrades do not backfill it. It panics if the embedded file is invalid,
// which unit tests rule out.
func DefaultManifest() *Manifest {
	manifest, err := parseManifest(defaultManifestYAML)
	if err != nil {
		panic(fmt.Sprintf("invalid built-in preset manifest: %v", err))
	}
	return manifest
}

// parseManifest decodes with viper, as config.Init does for presets.manifest,
// so the built-in file can be pasted there unchanged.
func parseManifest(data []byte) (*Manifest, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewReader(data)); err != nil {
		return nil, err
	}
	manifest := new(Manifest)
	if err := v.UnmarshalExact(manifest); err != nil {
		return nil, err
	}
	if err := manifest.Validate(); err != nil {
		return nil, err
	}
	return manifest, nil
}

//go:generate go tool mockgen -source=preset.go -destination=mocks/preset_repo_mock.go -package=mocks
type IPresetRepo interface {
	// SeedIfEmpty atomically inserts all presets only if registries and projects
	// are both empty. A concurrent seed's primary-key conflict is a no-op.
	SeedIfEmpty(ctx context.Context, manifest *Manifest) (applied bool, err error)
}
