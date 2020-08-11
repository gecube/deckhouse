locals {
  root_disk_size = lookup(var.providerClusterConfiguration.masterNodeGroup.instanceClass, "rootDiskSize", "")
  image_name = var.providerClusterConfiguration.masterNodeGroup.instanceClass.imageName
  flavor_name = var.providerClusterConfiguration.masterNodeGroup.instanceClass.flavorName
  security_group_names = concat([local.prefix], lookup(var.providerClusterConfiguration.masterNodeGroup.instanceClass, "additionalSecurityGroups", []))
}

module "network_security_info" {
  source = "../../../terraform-modules/network-security-info"
  prefix = local.prefix
  enabled = local.network_security
}

module "master" {
  source = "../../../terraform-modules/master"
  prefix = local.prefix
  node_index = var.nodeIndex
  cloud_config = var.cloudConfig
  root_disk_size = local.root_disk_size
  image_name = local.image_name
  flavor_name = local.flavor_name
  keypair_ssh_name = data.openstack_compute_keypair_v2.ssh.name
  network_port_ids = list(local.network_security ? openstack_networking_port_v2.master_internal_with_security[0].id : openstack_networking_port_v2.master_internal_without_security[0].id)
  floating_ip_network = data.openstack_networking_network_v2.external.name
}

module "kubernetes_data" {
  source = "../../../terraform-modules/kubernetes-data"
  prefix = local.prefix
  node_index = var.nodeIndex
  master_id = module.master.id
  volume_type = var.providerClusterConfiguration.masterNodeGroup.instanceClass.kubernetesDataVolumeType
}

module "security_groups" {
  source = "../../../terraform-modules/security-groups"
  security_group_names = local.security_group_names
  layout_security_group_ids = module.network_security_info.security_group_ids
  layout_security_group_names = module.network_security_info.security_group_names
}

data "openstack_compute_keypair_v2" "ssh" {
  name = local.prefix
}

data "openstack_networking_network_v2" "external" {
  name = local.external_network_name
}

data "openstack_networking_network_v2" "internal" {
  name = local.prefix
}

data "openstack_networking_subnet_v2" "internal" {
  name = local.prefix
}

resource "openstack_networking_port_v2" "master_internal_with_security" {
  count = local.network_security ? 1 : 0
  network_id = data.openstack_networking_network_v2.internal.id
  admin_state_up = "true"
  security_group_ids = module.security_groups.security_group_ids
  fixed_ip {
    subnet_id = data.openstack_networking_subnet_v2.internal.id
  }
  allowed_address_pairs {
    ip_address = local.pod_subnet_cidr
  }
}

resource "openstack_networking_port_v2" "master_internal_without_security" {
  count = local.network_security ? 0 : 1
  network_id = data.openstack_networking_network_v2.internal.id
  admin_state_up = "true"
  fixed_ip {
    subnet_id = data.openstack_networking_subnet_v2.internal.id
  }
}