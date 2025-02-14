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
	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"

	"github.com/deckhouse/deckhouse/go_lib/hooks/order_certificate"
)

var _ = order_certificate.RegisterOrderCertificateHook(
	[]order_certificate.OrderCertificateRequest{
		{
			Namespace:  "d8-user-authn",
			SecretName: "dex-tls",
			CommonName: "dex.d8-user-authn",
			SANs: []string{
				"dex.d8-user-authn",
				"dex.d8-user-authn.svc",
				order_certificate.ClusterDomainSAN("dex.d8-user-authn.svc"),
				order_certificate.PublicDomainSAN("dex"),
			},
			Usages: []certificatesv1beta1.KeyUsage{
				certificatesv1beta1.UsageDigitalSignature,
				certificatesv1beta1.UsageKeyEncipherment,
				certificatesv1beta1.UsageServerAuth,
			},
			ValueName:  "internal.dexTLS",
			ModuleName: "userAuthn",
		},
	},
)
