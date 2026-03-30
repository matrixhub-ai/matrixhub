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

package authz

import (
	"fmt"
	"regexp"
	"strings"
)

// Permission represents permission type, format is Resource.Action
type Permission string

func (p Permission) String() string {
	return string(p)
}

func (p Permission) IsWildcard() bool {
	return strings.Contains(string(p), "*")
}

// ToRegex converts wildcard permission to regex
func (p Permission) ToRegex() *regexp.Regexp {
	if !p.IsWildcard() {
		return nil
	}
	perm := strings.ReplaceAll(string(p), ".", "\\.")
	perm = strings.ReplaceAll(perm, "*", "[^.]+")
	perm = fmt.Sprintf("^%s$", perm)
	reg, _ := regexp.Compile(perm)
	return reg
}

// Match checks if permission matches (supports wildcard)
// User's permission p satisfies required permission
func (p Permission) Match(required Permission) bool {
	// Exact match
	if p == required {
		return true
	}

	// Wildcard match: user permission is wildcard, check if it matches required permission
	// Example: user has "model.*", needs "model.read" -> match
	pRegex := p.ToRegex()
	if pRegex != nil && pRegex.MatchString(required.String()) {
		return true
	}

	return false
}

// MatchPermissions checks if permission list has matching permission
func MatchPermissions(userPerms []Permission, required Permission) bool {
	for _, p := range userPerms {
		if p.Match(required) {
			return true
		}
	}
	return false
}

// Platform-level permissions
const (
	// User management
	UserGet           Permission = "user.get"            // View platform user list
	UserCreate        Permission = "user.create"         // Create platform user
	UserDelete        Permission = "user.delete"         // Delete platform user
	UserResetPassword Permission = "user.reset_password" // Reset user password
	UserAuthorize     Permission = "user.authorize"      // Authorize user (e.g., set as sysadmin)

	// Registry management
	RegistryGet    Permission = "registry.get"    // View registry list
	RegistryCreate Permission = "registry.create" // Create registry
	RegistryUpdate Permission = "registry.update" // Update registry
	RegistryDelete Permission = "registry.delete" // Delete registry

	// Sync policy management
	SyncGet    Permission = "sync.get"    // View sync policies
	SyncCreate Permission = "sync.create" // Create sync policy
	SyncUpdate Permission = "sync.update" // Update sync policy
	SyncDelete Permission = "sync.delete" // Delete sync policy

	// Access key management
	AccessKeyGet    Permission = "access_key.get"    // View access keys
	AccessKeyCreate Permission = "access_key.create" // Create access key
	AccessKeyDelete Permission = "access_key.delete" // Delete access key
)

// Project-level permissions
const (
	// Project member permissions
	MemberGet        Permission = "member.get"         // View project members
	MemberAdd        Permission = "member.add"         // Add project member
	MemberRemove     Permission = "member.remove"      // Remove project member
	MemberRoleUpdate Permission = "member.role_update" // Update member role

	// Project permissions
	ProjectGet    Permission = "project.get"    // View project
	ProjectCreate Permission = "project.create" // Create project
	ProjectUpdate Permission = "project.update" // Update project
	ProjectDelete Permission = "project.delete" // Delete project

	// Model permissions
	ModelGet     Permission = "model.get"     // View model list
	ModelPull    Permission = "model.pull"    // Pull model
	ModelPush    Permission = "model.push"    // Push model
	ModelDelete  Permission = "model.delete"  // Delete model
	ModelSetting Permission = "model.setting" // Update model settings

	// Dataset permissions
	DatasetGet    Permission = "dataset.get"    // View dataset list
	DatasetPull   Permission = "dataset.pull"   // Pull dataset
	DatasetPush   Permission = "dataset.push"   // Push dataset
	DatasetDelete Permission = "dataset.delete" // Delete dataset
)
