package hooks

import (
	"encoding/base64"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/deckhouse/deckhouse/testing/hooks"
)

var _ = Describe("Modules :: cloud-provider-openstack :: hooks :: openstack_cluster_configuration ::", func() {
	const (
		initValuesStringA = `
global:
  discovery": {}
cloudProviderOpenstack:
  internal:
    instances: {}
`
		initValuesStringB = `
global:
  discovery: {}
cloudProviderOpenstack:
  internal:
    instances: {}
  connection:
    authURL: https://test.tests.com:5000/v3/
    domainName: default
    tenantName: default
    username: jamie
    password: nein
    region: HetznerFinland
  externalNetworkNames: [public1, public2]
  internalNetworkNames: [int1, int2]
  podNetworkMode: DirectRouting
  instances:
    sshKeyPairName: my-ssh-keypair
    securityGroups:
    - security_group_1
    - security_group_2
  internalSubnet: "10.0.201.0/16"
  loadBalancer:
    subnetID: overrideSubnetID
`
	)

	var (
		stateACloudDiscoveryData = `
{
  "externalNetworkNames": [
    "external"
  ],
  "instances": {
    "securityGroups": [
      "default",
      "ssh-and-ping",
      "security_group_1"
    ]
  },
  "internalNetworkNames": [
    "internal"
  ],
  "podNetworkMode": "DirectRoutingWithPortSecurityEnabled",
  "zones": ["zone1", "zone2"],
  "loadBalancer": {
    "subnetID": "subnetID",
    "floatingNetworkID": "floatingNetworkID"
  }
}
`
		stateAClusterConfiguration = `
apiVersion: deckhouse.io/v1alpha1
kind: OpenStackClusterConfiguration
layout: Standard
standard:
  internalNetworkCIDR: 192.168.199.0/24
  internalNetworkDNSServers: ["8.8.8.8"]
  internalNetworkSecurity: true
  externalNetworkName: public
provider:
  authURL: https://cloud.flant.com/v3/
  domainName: Default
  tenantName: tenant-name
  username: user-name
  password: pa$$word
  region: HetznerFinland
`
		stateA = fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: d8-cluster-configuration
  namespace: kube-system
data:
  "cloud-provider-cluster-configuration.yaml": %s
  "cloud-provider-discovery-data.json": %s
`, base64.StdEncoding.EncodeToString([]byte(stateAClusterConfiguration)), base64.StdEncoding.EncodeToString([]byte(stateACloudDiscoveryData)))

		stateB = `
apiVersion: v1
kind: Secret
metadata:
 name: d8-provider-cluster-configuration
 namespace: kube-system
data: {}
`
	)

	f := HookExecutionConfigInit(initValuesStringA, `{}`)

	Context("Cluster has empty cloudProviderOpenstack and discovery data", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(f.KubeStateSet(stateA))
			f.RunHook()
		})

		It("Should fill values from discovery data", func() {
			Expect(f).To(ExecuteSuccessfully())
			connection := "cloudProviderOpenstack.internal.connection."
			Expect(f.ValuesGet(connection + "authURL").String()).To(Equal("https://cloud.flant.com/v3/"))
			Expect(f.ValuesGet(connection + "domainName").String()).To(Equal("Default"))
			Expect(f.ValuesGet(connection + "tenantName").String()).To(Equal("tenant-name"))
			Expect(f.ValuesGet(connection + "username").String()).To(Equal("user-name"))
			Expect(f.ValuesGet(connection + "password").String()).To(Equal("pa$$word"))
			Expect(f.ValuesGet(connection + "region").String()).To(Equal("HetznerFinland"))
			internal := "cloudProviderOpenstack.internal."
			Expect(f.ValuesGet(internal + "internalNetworkNames").String()).To(MatchYAML(`
[internal]
`))
			Expect(f.ValuesGet(internal + "externalNetworkNames").String()).To(MatchYAML(`
[external]
`))
			Expect(f.ValuesGet(internal + "zones").String()).To(MatchYAML(`
["zone1", "zone2"]
`))
			Expect(f.ValuesGet(internal + "podNetworkMode").String()).To(Equal("DirectRoutingWithPortSecurityEnabled"))
			Expect(f.ValuesGet(internal + "instances.securityGroups").String()).To(MatchYAML(`
[default, security_group_1, ssh-and-ping]
`))
			Expect(f.ValuesGet(internal + "loadBalancer").String()).To(MatchYAML(`
subnetID: "subnetID"
floatingNetworkID: "floatingNetworkID"
`))
		})
	})

	b := HookExecutionConfigInit(initValuesStringB, `{}`)
	Context("BeforeHelm", func() {
		BeforeEach(func() {
			b.BindingContexts.Set(BeforeHelmContext)
			b.RunHook()
		})

		It("Should fill values from cloudProviderOpenstack", func() {
			Expect(b).To(ExecuteSuccessfully())
			Expect(b.ValuesGet("cloudProviderOpenstack.internal").String()).To(MatchYAML(`
connection:
  authURL: https://test.tests.com:5000/v3/
  domainName: default
  tenantName: default
  username: jamie
  password: nein
  region: HetznerFinland
externalNetworkNames: [public1, public2]
internalNetworkNames: [int1, int2]
podNetworkMode: DirectRouting
instances:
  sshKeyPairName: my-ssh-keypair
  securityGroups:
  - security_group_1
  - security_group_2
zones: []
loadBalancer:
  subnetID: overrideSubnetID
`))
		})
	})

	Context("Fresh cluster", func() {
		BeforeEach(func() {
			b.BindingContexts.Set(b.KubeStateSet(""))
			b.RunHook()
		})
		It("Should fill values from cloudProviderOpenstack", func() {
			Expect(b).To(ExecuteSuccessfully())
			connection := "cloudProviderOpenstack.internal.connection."
			Expect(b.ValuesGet(connection + "authURL").String()).To(Equal("https://test.tests.com:5000/v3/"))
			Expect(b.ValuesGet(connection + "domainName").String()).To(Equal("default"))
			Expect(b.ValuesGet(connection + "tenantName").String()).To(Equal("default"))
			Expect(b.ValuesGet(connection + "username").String()).To(Equal("jamie"))
			Expect(b.ValuesGet(connection + "password").String()).To(Equal("nein"))
			Expect(b.ValuesGet(connection + "region").String()).To(Equal("HetznerFinland"))
			internal := "cloudProviderOpenstack.internal."
			Expect(b.ValuesGet(internal + "internalNetworkNames").String()).To(MatchYAML(`
[int1, int2]
`))
			Expect(b.ValuesGet(internal + "externalNetworkNames").String()).To(MatchYAML(`
[public1, public2]
`))
			Expect(b.ValuesGet(internal + "zones").String()).To(MatchYAML("[]"))
			Expect(b.ValuesGet(internal + "podNetworkMode").String()).To(Equal("DirectRouting"))
			Expect(b.ValuesGet(internal + "instances.securityGroups").String()).To(MatchYAML(`
[security_group_1, security_group_2]
`))
			Expect(b.ValuesGet(internal + "loadBalancer").String()).To(MatchYAML(`
subnetID: overrideSubnetID
`))
		})

		Context("Cluster has cloudProviderOpenstack and discovery data", func() {
			BeforeEach(func() {
				b.BindingContexts.Set(b.KubeStateSet(stateA))
				b.RunHook()
			})

			It("Should merge values from cloudProviderOpenstack and discovery data", func() {
				Expect(b).To(ExecuteSuccessfully())
				connection := "cloudProviderOpenstack.internal.connection."
				Expect(b.ValuesGet(connection + "authURL").String()).To(Equal("https://test.tests.com:5000/v3/"))
				Expect(b.ValuesGet(connection + "domainName").String()).To(Equal("default"))
				Expect(b.ValuesGet(connection + "tenantName").String()).To(Equal("default"))
				Expect(b.ValuesGet(connection + "username").String()).To(Equal("jamie"))
				Expect(b.ValuesGet(connection + "password").String()).To(Equal("nein"))
				Expect(b.ValuesGet(connection + "region").String()).To(Equal("HetznerFinland"))
				internal := "cloudProviderOpenstack.internal."
				Expect(b.ValuesGet(internal + "internalNetworkNames").String()).To(MatchYAML(`
[int1, int2, internal]
`))
				Expect(b.ValuesGet(internal + "externalNetworkNames").String()).To(MatchYAML(`
[external, public1, public2]
`))
				Expect(b.ValuesGet(internal + "zones").String()).To(MatchYAML(`
["zone1", "zone2"]
`))
				Expect(b.ValuesGet(internal + "podNetworkMode").String()).To(Equal("DirectRouting"))
				Expect(b.ValuesGet(internal + "instances.securityGroups").String()).To(MatchYAML(`
[default, security_group_1, security_group_2, ssh-and-ping]
`))
				Expect(b.ValuesGet(internal + "loadBalancer").String()).To(MatchYAML(`
subnetID: overrideSubnetID
floatingNetworkID: floatingNetworkID
`))
			})
		})
	})

	Context("Cluster has cloudProviderOpenstack and empty discovery data", func() {
		BeforeEach(func() {
			b.BindingContexts.Set(b.KubeStateSet(stateB))
			b.RunHook()
		})

		It("Should fill values from cloudProviderOpenstack", func() {
			Expect(b).To(ExecuteSuccessfully())
			connection := "cloudProviderOpenstack.internal.connection."
			Expect(b.ValuesGet(connection + "authURL").String()).To(Equal("https://test.tests.com:5000/v3/"))
			Expect(b.ValuesGet(connection + "domainName").String()).To(Equal("default"))
			Expect(b.ValuesGet(connection + "tenantName").String()).To(Equal("default"))
			Expect(b.ValuesGet(connection + "username").String()).To(Equal("jamie"))
			Expect(b.ValuesGet(connection + "password").String()).To(Equal("nein"))
			Expect(b.ValuesGet(connection + "region").String()).To(Equal("HetznerFinland"))
			internal := "cloudProviderOpenstack.internal."
			Expect(b.ValuesGet(internal + "internalNetworkNames").String()).To(MatchYAML(`
[int1, int2]
`))
			Expect(b.ValuesGet(internal + "externalNetworkNames").String()).To(MatchYAML(`
[public1, public2]
`))
			Expect(b.ValuesGet(internal + "podNetworkMode").String()).To(Equal("DirectRouting"))
			Expect(b.ValuesGet(internal + "instances.securityGroups").String()).To(MatchYAML(`
[security_group_1, security_group_2]
`))
			Expect(b.ValuesGet(internal + "loadBalancer").String()).To(MatchYAML(`
subnetID: overrideSubnetID
`))
		})
	})
})