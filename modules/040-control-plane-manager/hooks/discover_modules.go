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
	"github.com/flant/shell-operator/pkg/kube_events_manager/types"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = sdk.RegisterFunc(&go_hook.HookConfig{
	Queue: moduleQueue,
	Kubernetes: []go_hook.KubernetesConfig{
		{
			Name:       "auth-cm",
			ApiVersion: "v1",
			Kind:       "ConfigMap",
			NamespaceSelector: &types.NamespaceSelector{
				NameSelector: &types.NameSelector{
					MatchNames: []string{"d8-user-authn", "d8-user-authz"},
				},
			},
			LabelSelector: &v1.LabelSelector{
				MatchLabels: map[string]string{
					"control-plane-configurator": "",
				},
			},
			FilterFunc: discoveryFilterSecrets,
		},
	},
}, handleAuthDiscoveryModules)

func discoveryFilterSecrets(unstructured *unstructured.Unstructured) (go_hook.FilterResult, error) {
	var cm corev1.ConfigMap

	err := sdk.FromUnstructured(unstructured, &cm)
	if err != nil {
		return nil, err
	}

	return discoveryCM{Namespace: cm.Namespace, Data: cm.Data}, nil
}

type discoveryCM struct {
	Namespace string
	Data      map[string]string
}

func handleAuthDiscoveryModules(input *go_hook.HookInput) error {
	snap := input.Snapshots["auth-cm"]
	var authZData, authNData map[string]string

	for _, s := range snap {
		cm := s.(discoveryCM)
		switch cm.Namespace {
		case "d8-user-authn":
			authNData = cm.Data

		case "d8-user-authz":
			authZData = cm.Data
		}
	}

	const (
		userAuthzWebhookURLPath = "controlPlaneManager.apiserver.authz.webhookURL"
		userAuthzWebhookCAPath  = "controlPlaneManager.apiserver.authz.webhookCA"

		userAuthnOIDCIssuerURLPath     = "controlPlaneManager.apiserver.authn.oidcIssuerURL"
		userAuthnOIDCIssuerAddressPath = "controlPlaneManager.apiserver.authn.oidcIssuerAddress"
		userAuthnOIDCIssuerCAPath      = "controlPlaneManager.apiserver.authn.oidcCA"
	)

	authzWebhookURLExists := input.ConfigValues.Exists(userAuthzWebhookURLPath)
	authzWebhookCAExists := input.ConfigValues.Exists(userAuthzWebhookCAPath)

	authnOIDCIssuerExists := input.ConfigValues.Exists(userAuthnOIDCIssuerURLPath)
	authnOIDCCAExists := input.ConfigValues.Exists(userAuthnOIDCIssuerCAPath)

	if !authzWebhookURLExists && !authzWebhookCAExists {
		// nothing was configured by hand
		if len(authZData) > 0 {
			input.Values.Set(userAuthzWebhookURLPath, authZData["url"])
			input.Values.Set(userAuthzWebhookCAPath, authZData["ca"])
		} else {
			input.Values.Remove(userAuthzWebhookURLPath)
			input.Values.Remove(userAuthzWebhookCAPath)
		}
	}

	if !authnOIDCIssuerExists && !authnOIDCCAExists {
		// nothing was configured by hand
		if len(authNData) > 0 {
			input.Values.Set(userAuthnOIDCIssuerURLPath, authNData["oidcIssuerURL"])
			input.Values.Set(userAuthnOIDCIssuerCAPath, input.Values.Get("global.discovery.kubernetesCA").String())
			if address, ok := authNData["oidcIssuerAddress"]; ok {
				input.Values.Set(userAuthnOIDCIssuerAddressPath, address)
			}
		} else {
			input.Values.Remove(userAuthnOIDCIssuerURLPath)
			input.Values.Remove(userAuthnOIDCIssuerCAPath)
			input.Values.Remove(userAuthnOIDCIssuerAddressPath)
		}
	}

	return nil
}
