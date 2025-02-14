---
title: "Cloud provider — VMware vSphere: Preparing environment"
---

## Configuring vSphere

The vSphere CLI called [govc](https://github.com/vmware/govmomi/tree/master/govc#installation) is designed to configure vSphere.

### Setting up govc

```shell
export GOVC_URL=n-cs-5.hq.li.corp.kavvas.com
export GOVC_USERNAME=ewwefsadfsda
export GOVC_PASSWORD=weqrfweqfeds
export GOVC_INSECURE=1
```

### Creating tags and tag categories

Note that vSphere lacks the concept of "regions" and "zones", so tags are used for the availability zones differentiation.
For example, here is how you can create two regions with two availability zones in each region:

```shell
govc tags.category.create -d "Kubernetes region" k8s-region
govc tags.category.create -d "Kubernetes zone" k8s-zone
govc tags.create -d "Kubernetes Region X1" -c k8s-region k8s-region-x1
govc tags.create -d "Kubernetes Region X2" -c k8s-region k8s-region-x2
govc tags.create -d "Kubernetes Zone X1-A" -c k8s-zone k8s-zone-x1-a
govc tags.create -d "Kubernetes Zone X1-B" -c k8s-zone k8s-zone-x1-b
govc tags.create -d "Kubernetes Zone X2-A" -c k8s-zone k8s-zone-x2-a
govc tags.create -d "Kubernetes Zone X2-B" -c k8s-zone k8s-zone-x2-b
```

The created tag categories must be specified in the `VsphereClusterConfiguration` in `.spec.provider`.

The *region* tags are attached to the Datacenter:

```shell
govc tags.attach -c k8s-region k8s-region-x1 /X1
```

The *zone* tags are attached to the Cluster and Datastores:

```shell
govc tags.attach -c k8s-zone k8s-zone-x1-a /X1/host/x1_cluster_prod
govc tags.attach -c k8s-zone k8s-zone-x1-a /X1/datastore/x1_lun_1
```

### Permissions

You need to create a Role with the permissions specified below and attach it to one or more Datacenters where the Kubernetes cluster will be deployed.

Note that we do not provide instructions for creating a user due to a wide variety of SSOs used with vSphere.

```shell
govc role.create kubernetes Datastore.AllocateSpace Datastore.Browse Datastore.FileManagement Global.GlobalTag Global.SystemTag InventoryService.Tagging.AttachTag InventoryService.Tagging.CreateCategory InventoryService.Tagging.CreateTag InventoryService.Tagging.DeleteCategory InventoryService.Tagging.DeleteTag InventoryService.Tagging.EditCategory InventoryService.Tagging.EditTag InventoryService.Tagging.ModifyUsedByForCategory InventoryService.Tagging.ModifyUsedByForTag Network.Assign Resource.AssignVMToPool Resource.ColdMigrate Resource.HotMigrate Resource.CreatePool Resource.DeletePool Resource.RenamePool Resource.EditPool Resource.MovePool StorageProfile.View System.Anonymous System.Read System.View VirtualMachine.Config.AddExistingDisk VirtualMachine.Config.AddNewDisk VirtualMachine.Config.AddRemoveDevice VirtualMachine.Config.AdvancedConfig VirtualMachine.Config.Annotation VirtualMachine.Config.CPUCount VirtualMachine.Config.ChangeTracking VirtualMachine.Config.DiskExtend VirtualMachine.Config.DiskLease VirtualMachine.Config.EditDevice VirtualMachine.Config.HostUSBDevice VirtualMachine.Config.ManagedBy VirtualMachine.Config.Memory VirtualMachine.Config.MksControl VirtualMachine.Config.QueryFTCompatibility VirtualMachine.Config.QueryUnownedFiles VirtualMachine.Config.RawDevice VirtualMachine.Config.ReloadFromPath VirtualMachine.Config.RemoveDisk VirtualMachine.Config.Rename VirtualMachine.Config.ResetGuestInfo VirtualMachine.Config.Resource VirtualMachine.Config.Settings VirtualMachine.Config.SwapPlacement VirtualMachine.Config.ToggleForkParent VirtualMachine.Config.UpgradeVirtualHardware VirtualMachine.GuestOperations.Execute VirtualMachine.GuestOperations.Modify VirtualMachine.GuestOperations.ModifyAliases VirtualMachine.GuestOperations.Query VirtualMachine.GuestOperations.QueryAliases VirtualMachine.Hbr.ConfigureReplication VirtualMachine.Hbr.MonitorReplication VirtualMachine.Hbr.ReplicaManagement VirtualMachine.Interact.AnswerQuestion VirtualMachine.Interact.Backup VirtualMachine.Interact.ConsoleInteract VirtualMachine.Interact.CreateScreenshot VirtualMachine.Interact.CreateSecondary VirtualMachine.Interact.DefragmentAllDisks VirtualMachine.Interact.DeviceConnection VirtualMachine.Interact.DisableSecondary VirtualMachine.Interact.DnD VirtualMachine.Interact.EnableSecondary VirtualMachine.Interact.GuestControl VirtualMachine.Interact.MakePrimary VirtualMachine.Interact.Pause VirtualMachine.Interact.PowerOff VirtualMachine.Interact.PowerOn VirtualMachine.Interact.PutUsbScanCodes VirtualMachine.Interact.Record VirtualMachine.Interact.Replay VirtualMachine.Interact.Reset VirtualMachine.Interact.SESparseMaintenance VirtualMachine.Interact.SetCDMedia VirtualMachine.Interact.SetFloppyMedia VirtualMachine.Interact.Suspend VirtualMachine.Interact.TerminateFaultTolerantVM VirtualMachine.Interact.ToolsInstall VirtualMachine.Interact.TurnOffFaultTolerance VirtualMachine.Inventory.Create VirtualMachine.Inventory.CreateFromExisting VirtualMachine.Inventory.Delete VirtualMachine.Inventory.Move VirtualMachine.Inventory.Register VirtualMachine.Inventory.Unregister VirtualMachine.Namespace.Event VirtualMachine.Namespace.EventNotify VirtualMachine.Namespace.Management VirtualMachine.Namespace.ModifyContent VirtualMachine.Namespace.Query VirtualMachine.Namespace.ReadContent VirtualMachine.Provisioning.Clone VirtualMachine.Provisioning.CloneTemplate VirtualMachine.Provisioning.CreateTemplateFromVM VirtualMachine.Provisioning.Customize VirtualMachine.Provisioning.DeployTemplate VirtualMachine.Provisioning.DiskRandomAccess VirtualMachine.Provisioning.DiskRandomRead VirtualMachine.Provisioning.FileRandomAccess VirtualMachine.Provisioning.GetVmFiles VirtualMachine.Provisioning.MarkAsTemplate VirtualMachine.Provisioning.MarkAsVM VirtualMachine.Provisioning.ModifyCustSpecs VirtualMachine.Provisioning.PromoteDisks VirtualMachine.Provisioning.PutVmFiles VirtualMachine.Provisioning.ReadCustSpecs VirtualMachine.State.CreateSnapshot VirtualMachine.State.RemoveSnapshot VirtualMachine.State.RenameSnapshot VirtualMachine.State.RevertToSnapshot

govc permissions.set  -principal username -role kubernetes /datacenter
```

## Infrastructure

### Networking
A VLAN with DHCP and Internet access is required for the running cluster:
* If the VLAN is public (white addresses), then you have to create a second network to deploy cluster nodes (DHCP is not needed in this network);
* If the VLAN is private (gray addresses), then this network can be used for cluster nodes;

### Inbound traffic
* You can use an internal load balancer (if present) and direct traffic directly to the front nodes of the cluster.
* If there is no load balancer, you can use MetalLB in BGP mode to organize fault-tolerant load balancers (recommended). In this case, front nodes of the cluster will have two interfaces. For this, you will need:
  * A dedicated VLAN for traffic exchange between BGP routers and MetalLB. This VLAN must have DHCP and Internet access;
  * IP addresses of BGP routers;
  * ASN — the AS number on the BGP router;
  * ASN — the AS number in the cluster;
  * A range to announce addresses from;

### Using the data store
Various types of storage can be used in the cluster; for the minimum configuration, you will need:
* Datastore for provisioning PersistentVolumes to the Kubernetes cluster;
* Datastore for provisioning root disks for the VMs (it can be the same Datastore as for PersistentVolume).
