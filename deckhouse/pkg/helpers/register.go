package helpers

import (
	"gopkg.in/alecthomas/kingpin.v2"

	sh_app "github.com/flant/shell-operator/pkg/app"

	"flant/deckhouse/pkg/helpers/aws"
	"flant/deckhouse/pkg/helpers/fnv"
	"flant/deckhouse/pkg/helpers/openstack"
	"flant/deckhouse/pkg/helpers/unit"
	"flant/deckhouse/pkg/helpers/vsphere"
)

func DefineHelperCommands(kpApp *kingpin.Application) {
	helpersCommand := sh_app.CommandWithDefaultUsageTemplate(kpApp, "helper", "Deckhouse helpers.")

	fnvCommand := helpersCommand.Command("fnv", "Section for command related tp fnv encoding and decoding.")
	fnvEncodeCommand := fnvCommand.Command("encode", "Encode input in FNV styled string.")
	fnvEncodeInput := fnvEncodeCommand.Arg("input", "String to encode").Required().String()
	fnvEncodeCommand.Action(func(c *kingpin.ParseContext) error {
		return fnv.Encode(*fnvEncodeInput)
	})

	unitCommand := helpersCommand.Command("unit", "Unit related methods.")
	unitConvertCommand := unitCommand.Command("convert", "Convert units.")
	unitConvertMode := unitConvertCommand.Flag("mode", "Mode of unit converter").Enum("duration", "kube-resource-unit")
	unitConvertCommand.Action(func(c *kingpin.ParseContext) error {
		return unit.Convert(*unitConvertMode)
	})

	awsCommand := helpersCommand.Command("aws", "AWS helpers.")
	awsMapZoneToSubnetsCommand := awsCommand.Command("map-zone-to-subnets", "Map zones to subnets.")
	awsMapZoneToSubnetsCommand.Action(func(c *kingpin.ParseContext) error {
		return aws.MapZoneToSubnets()
	})

	openstackCommand := helpersCommand.Command("openstack", "OpenStack helpers.")
	openstackGetVolumeTypes := openstackCommand.Command("get-volume-types", "Get volume types.")
	openstackGetVolumeTypes.Action(func(c *kingpin.ParseContext) error {
		return openstack.GetVolumeTypes()
	})

	vsphereCommand := helpersCommand.Command("vsphere", "VSphere helpers.")
	vsphereGetZonesDatastores := vsphereCommand.Command("get-zones-datastores", "Get zones datastores.")
	vsphereGetZonesDatastores.Action(func(c *kingpin.ParseContext) error {
		return vsphere.GetZonesDatastores()
	})
}
