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
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

// Limits follow the database columns and the create APIs, so a valid manifest
// cannot fail part-way through seeding.
const (
	maxNameLength      = 64
	maxURLLength       = 255
	maxModelNameLength = 251
)

var (
	projectNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9]$`)
	// Model names and upstream organizations follow Hugging Face repository
	// naming, which also keeps them from being "." or "..".
	modelNamePattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	organizationPattern = modelNamePattern
	reservedModelNames  = []string{"models", "datasets", "spaces", "resolve"}
	labelCategories     = []string{"library", "license", "task", "language", "other"}
)

// Validate reports every problem in the manifest, each prefixed with its YAML
// path, such as models[3].project.
func (m *Manifest) Validate() error {
	var errs []error
	report := func(path string, err error) {
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", path, err))
		}
	}

	registries := make(map[string]bool, len(m.Registries))
	for i, spec := range m.Registries {
		path := fmt.Sprintf("registries[%d]", i)
		if err := validateName(spec.Name, maxNameLength, nil); err != nil {
			report(path+".name", err)
		} else if registries[spec.Name] {
			report(path+".name", fmt.Errorf("duplicate registry %q", spec.Name))
		}
		registries[spec.Name] = true
		report(path+".url", validateURL(spec.URL))
	}

	projects := make(map[string]bool, len(m.Projects))
	for i, spec := range m.Projects {
		path := fmt.Sprintf("projects[%d]", i)
		if err := validateName(spec.Name, maxNameLength, projectNamePattern); err != nil {
			report(path+".name", err)
		} else if projects[spec.Name] {
			report(path+".name", fmt.Errorf("duplicate project %q", spec.Name))
		}
		projects[spec.Name] = true
		if spec.Organization != "" {
			report(path+".organization", validateName(spec.Organization, maxNameLength, organizationPattern))
		}
		report(path+".registry", validateReference(registries, spec.Registry))
	}

	models := make(map[string]bool, len(m.Models))
	for i, spec := range m.Models {
		path := fmt.Sprintf("models[%d]", i)
		report(path+".project", validateReference(projects, spec.Project))
		// Both databases compare model names within a project case-insensitively.
		key := spec.Project + "/" + strings.ToLower(spec.Name)
		if err := validateModelName(spec.Name); err != nil {
			report(path+".name", err)
		} else if models[key] {
			report(path+".name", fmt.Errorf("duplicate model %q in project %q (names are case-insensitive)",
				spec.Name, spec.Project))
		}
		models[key] = true
		if spec.Size < 0 {
			report(path+".size", errors.New("must not be negative"))
		}
		if spec.ParameterCount < 0 {
			report(path+".parameterCount", errors.New("must not be negative"))
		}

		labels := make(map[string]bool, len(spec.Labels))
		for j, label := range spec.Labels {
			labelPath := fmt.Sprintf("%s.labels[%d]", path, j)
			if !slices.Contains(labelCategories, label.Category) {
				report(labelPath+".category", fmt.Errorf("must be one of: %s", strings.Join(labelCategories, ", ")))
			}
			// Label lookups ignore case, so case variants resolve to the same label.
			key := label.Category + ":" + strings.ToLower(label.Name)
			if err := validateName(label.Name, maxNameLength, nil); err != nil {
				report(labelPath+".name", err)
			} else if labels[key] {
				report(labelPath+".name", fmt.Errorf("duplicate label %s:%s (names are case-insensitive)",
					label.Category, label.Name))
			}
			labels[key] = true
		}
	}
	return errors.Join(errs...)
}

func validateName(value string, maxLength int, pattern *regexp.Regexp) error {
	switch {
	case strings.TrimSpace(value) == "":
		return errors.New("is required")
	case utf8.RuneCountInString(value) > maxLength:
		return fmt.Errorf("must be at most %d characters", maxLength)
	case pattern != nil && !pattern.MatchString(value):
		return fmt.Errorf("must match %s", pattern)
	}
	return nil
}

func validateModelName(name string) error {
	if err := validateName(name, maxModelNameLength, modelNamePattern); err != nil {
		return err
	}
	if slices.Contains(reservedModelNames, name) {
		return fmt.Errorf("%q is reserved", name)
	}
	return nil
}

func validateURL(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("is required")
	}
	if utf8.RuneCountInString(value) > maxURLLength {
		return fmt.Errorf("must be at most %d characters", maxURLLength)
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errors.New("must be an absolute http or https URL")
	}
	return nil
}

func validateReference(defined map[string]bool, name string) error {
	if name == "" {
		return errors.New("is required")
	}
	if !defined[name] {
		return fmt.Errorf("%q is not defined in the manifest", name)
	}
	return nil
}
