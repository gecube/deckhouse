package hooks

import (
	"fmt"
	"strings"

	"github.com/blang/semver"
	"github.com/flant/addon-operator/pkg/module_manager/go_hook"
	"github.com/flant/addon-operator/sdk"

	"github.com/deckhouse/deckhouse/go_lib/dependency"
)

var _ = sdk.RegisterFunc(&go_hook.HookConfig{
	OnBeforeHelm: &go_hook.OrderedConfig{
		Order: 20,
	},
}, dependency.WithExternalDependencies(handleGlobalValuesAndKubectl))

func handleGlobalValuesAndKubectl(input *go_hook.HookInput, dc dependency.Container) error {
	var (
		cloudProvider           = "none"
		controlPlaneVersion     semver.Version
		clusterType             = "Cloud"
		terraformManagerEnabled bool
	)

	modules := input.Values.Get("global.enabledModules").Array()
	if modules == nil {
		return fmt.Errorf("got nil global.enabledModules")
	}
	for _, module := range modules {
		moduleName := module.String()
		if strings.HasPrefix(moduleName, "cloud-provider-") {
			cloudProvider = strings.TrimPrefix(moduleName, "cloud-provider-")
		}
		if moduleName == "terraform-manager" {
			terraformManagerEnabled = true
		}
	}

	k8, err := dc.GetK8sClient()
	if err != nil {
		return fmt.Errorf("can't init Kubernetes client: %v", err)
	}
	version, err := k8.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("can't get Kubernetes version: %v", err)
	}
	serverVersion := version.String()
	controlPlaneVersion, err = semver.Make(serverVersion[1:])
	if err != nil {
		return fmt.Errorf("can't parse Kubernetes version: %v", err)
	}

	if input.Values.Exists("global.clusterConfiguration") {
		clusterType = input.Values.Get("global.clusterConfiguration.clusterType").String()
		staticNodesCount, ok := input.Values.GetOk("flantPricing.internal.nodeStats.staticNodesCount")
		if !ok {
			return fmt.Errorf("waiting for `internal.nodeStats.staticNodesCount` to be defined")
		}
		if (clusterType == "Static" && cloudProvider != "none") || (clusterType == "Cloud" && staticNodesCount.Int() > 0) {
			clusterType = "Hybrid"
		}
	}

	if input.Values.Exists("flantPricing.clusterType") {
		clusterType = input.Values.Get("flantPricing.clusterType").String()
	}

	input.Values.Set("flantPricing.internal.cloudProvider", cloudProvider)
	input.Values.Set("flantPricing.internal.controlPlaneVersion",
		fmt.Sprintf("%d.%d", controlPlaneVersion.Major, controlPlaneVersion.Minor))

	input.Values.Set("flantPricing.internal.clusterType", clusterType)
	input.Values.Set("flantPricing.internal.terraformManagerEnabled", terraformManagerEnabled)

	return nil
}