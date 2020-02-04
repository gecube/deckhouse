/*

User-stories:
1. There are services with label `prometheus.deckhous.io/alertmanager: <prometheus_instance>. Hook must discover them and store to values `prometheus.internal.alertmanagers` in format {"<prometheus_instance>": [{<service_description>}, ...], ...}.
   There is optional annotation `prometheus.deckhouse.io/alertmanager-path-prefix` with default value "/". It must be stored in service description.

*/

package hooks

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/deckhouse/deckhouse/testing/hooks"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "")
}

var _ = Describe("Prometheus hooks :: alertmanager discovery ::", func() {
	const (
		initValuesString       = `{"prometheus": {"internal": {}}}`
		initConfigValuesString = `{}`
	)

	const (
		stateNonSpecialServices = `
---
apiVersion: v1
kind: Service
metadata:
  name: some-svc-1
  namespace: some-ns-1
---
apiVersion: v1
kind: Service
metadata:
  name: some-svc-2
  namespace: some-ns-2
`

		stateSpecialServicesAlpha = `
---
apiVersion: v1
kind: Service
metadata:
  name: mysvc1
  namespace: myns1
  labels:
    prometheus.deckhouse.io/alertmanager: alphaprom
  annotations:
    prometheus.deckhouse.io/alertmanager-path-prefix: /myprefix/
spec:
  ports:
  - port: 81
`
		stateSpecialServicesBeta = `
---
apiVersion: v1
kind: Service
metadata:
  name: mysvc2
  namespace: myns2
  labels:
    prometheus.deckhouse.io/alertmanager: betaprom
spec:
  ports:
  - port: 82
---
apiVersion: v1
kind: Service
metadata:
  name: mysvc3
  namespace: myns3
  labels:
    prometheus.deckhouse.io/alertmanager: betaprom
spec:
  ports:
  - port: 83
`
	)

	f := HookExecutionConfigInit(initValuesString, initConfigValuesString)

	Context("Cluster has non-special services", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(f.KubeStateSet(stateNonSpecialServices))
			f.RunHook()
		})

		It("snapshots must be empty; prometheus.internal.alertmanagers must be '{}'", func() {
			Expect(f).To(ExecuteSuccessfully())
			Expect(f.BindingContexts.Get("0.snapshots.alertmanager_services").Array()).To(BeEmpty())
			Expect(f.ValuesGet("prometheus.internal.alertmanagers").String()).To(Equal("{}"))
		})
	})

	Context("Cluster has special service", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(f.KubeStateSet(stateNonSpecialServices + stateSpecialServicesAlpha))
			f.RunHook()
		})

		It(`prometheus.internal.alertmanagers must be '{"alphaprom":[{"name":"mysvc1","namespace":"myns1","pathPrefix":"/myprefix/","port":81}]}'`, func() {
			Expect(f).To(ExecuteSuccessfully())
			Expect(f.ValuesGet("prometheus.internal.alertmanagers").String()).To(Equal(`{"alphaprom":[{"name":"mysvc1","namespace":"myns1","pathPrefix":"/myprefix/","port":81}]}`))
		})

		Context("Two more special services added", func() {
			BeforeEach(func() {
				f.BindingContexts.Set(f.KubeStateSet(stateNonSpecialServices + stateSpecialServicesAlpha + stateSpecialServicesBeta))
				f.RunHook()
			})

			It(`prometheus.internal.alertmanagers must be '{"alphaprom":[{"name":"mysvc1","namespace":"myns1","pathPrefix":"/myprefix/","port":81}],"betaprom":[{"name":"mysvc2","namespace":"myns2","pathPrefix":"/","port":82},{"name":"mysvc3","namespace":"myns3","pathPrefix":"/","port":83}]}'`, func() {
				Expect(f).To(ExecuteSuccessfully())
				Expect(f.ValuesGet("prometheus.internal.alertmanagers").String()).To(Equal(`{"alphaprom":[{"name":"mysvc1","namespace":"myns1","pathPrefix":"/myprefix/","port":81}],"betaprom":[{"name":"mysvc2","namespace":"myns2","pathPrefix":"/","port":82},{"name":"mysvc3","namespace":"myns3","pathPrefix":"/","port":83}]}`))
			})
		})

	})
})