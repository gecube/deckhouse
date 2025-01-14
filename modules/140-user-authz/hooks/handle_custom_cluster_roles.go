/*
Copyright 2021 Flant CJSC

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

package hooks

import (
	"github.com/flant/addon-operator/pkg/module_manager/go_hook"
	"github.com/flant/addon-operator/sdk"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/deckhouse/deckhouse/modules/140-user-authz/hooks/internal"
)

const (
	ccrSnapshot = "custom_cluster_roles"
)

type CustomClusterRole struct {
	Name string
	Role string
}

type roleNamesSet map[string]struct{}

func (r roleNamesSet) setRole(roleName string) {
	r[roleName] = struct{}{}
}

func (r roleNamesSet) convertToSlice() []string {
	v := make([]string, 0, len(r))
	for k := range r {
		v = append(v, k)
	}
	return v
}

func applyCustomClusterRoleFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	ccr := &CustomClusterRole{}

	role := obj.GetAnnotations()["user-authz.deckhouse.io/access-level"]
	switch role {
	case "User", "PrivilegedUser", "Editor", "Admin", "ClusterEditor", "ClusterAdmin":
		ccr.Name = obj.GetName()
		ccr.Role = role
	default:
		return nil, nil
	}
	return ccr, nil
}

var _ = sdk.RegisterFunc(&go_hook.HookConfig{
	Queue: internal.Queue(ccrSnapshot),
	Kubernetes: []go_hook.KubernetesConfig{
		{
			Name:       ccrSnapshot,
			ApiVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
			FilterFunc: applyCustomClusterRoleFilter,
		},
	},
}, customClusterRolesHandler)

func customClusterRolesHandler(input *go_hook.HookInput) error {
	type internalValuesCustomClusterRoles struct {
		User           []string `json:"user"`
		PrivilegedUser []string `json:"privilegedUser"`
		Editor         []string `json:"editor"`
		Admin          []string `json:"admin"`
		ClusterEditor  []string `json:"clusterEditor"`
		ClusterAdmin   []string `json:"clusterAdmin"`
	}

	userRoleNames := &roleNamesSet{}
	privilegedUserRoleNames := &roleNamesSet{}
	editorRoleNames := &roleNamesSet{}
	adminRoleNames := &roleNamesSet{}
	clusterEditorRoleNames := &roleNamesSet{}
	clusterAdminRoleNames := &roleNamesSet{}

	snapshots := input.Snapshots[ccrSnapshot]

	for _, snapshot := range snapshots {
		if snapshot == nil {
			continue
		}
		customClusterRole := snapshot.(*CustomClusterRole)
		switch customClusterRole.Role {
		case "User":
			userRoleNames.setRole(customClusterRole.Name)
			fallthrough
		case "PrivilegedUser":
			privilegedUserRoleNames.setRole(customClusterRole.Name)
			fallthrough
		case "Editor":
			editorRoleNames.setRole(customClusterRole.Name)
			fallthrough
		case "Admin":
			adminRoleNames.setRole(customClusterRole.Name)
			fallthrough
		case "ClusterEditor":
			clusterEditorRoleNames.setRole(customClusterRole.Name)
			fallthrough
		case "ClusterAdmin":
			clusterAdminRoleNames.setRole(customClusterRole.Name)
		}
	}

	input.Values.Set("userAuthz.internal.customClusterRoles", internalValuesCustomClusterRoles{
		User:           userRoleNames.convertToSlice(),
		PrivilegedUser: privilegedUserRoleNames.convertToSlice(),
		Editor:         editorRoleNames.convertToSlice(),
		Admin:          adminRoleNames.convertToSlice(),
		ClusterEditor:  clusterEditorRoleNames.convertToSlice(),
		ClusterAdmin:   clusterAdminRoleNames.convertToSlice(),
	})

	return nil
}
