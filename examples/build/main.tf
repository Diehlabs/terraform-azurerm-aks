provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "example" {
  name     = var.rg_name
  location = local.tags.region
  tags     = local.tags
}

resource "tls_private_key" "example" {
  algorithm = "RSA"
  rsa_bits  = 2048
}

resource "local_file" "ssh_private_key" {
  content         = tls_private_key.example.private_key_pem
  filename        = "adminuser_rsa.key"
  file_permission = "0600"
}

resource "local_file" "kubeconfig" {
  content         = module.aks_cluster.kube_admin_config
  filename        = "kubeconfig"
  file_permission = "0600"
  depends_on = [
    module.aks_cluster
  ]
}

# data "azurerm_virtual_network" "example" {
#   name                = "westus-dev-aks-vnet"
#   resource_group_name = "westus-dev-aks-rg"
# }

# data "azurerm_log_analytics_workspace" "example" {
#   name                = "<laws-name>"
#   resource_group_name = "<laws-rg-name>"
# }

module "aks_cluster" {
  source                    = "../.."
  tags                      = local.tags
  subnet_id                 = data.terraform_remote_state.devtest_network.outputs.subnets["terratest"].id
  docker_bridge_cidr        = "192.168.0.1/16"
  dns_service_ip            = "172.16.100.126"
  service_cidr              = "172.16.100.0/25"
  node_count                = var.node_count
  kubernetes_version_number = "1.21.7"
  location                  = var.tags.region
  max_pods                  = 31
  linux_profile = {
    username = "adminuser"
    sshkey   = tls_private_key.example.public_key_openssh
  }
  msi_id            = var.msi_id                               #data.terraform_remote_state.devtest_infra.outputs.identity.id
  cluster_admin_ids = [data.azuread_group.kube_admins.object_id]
  # laws                = data.azurerm_log_analytics_workspace.example
  private_dns_zone_id = "/subscriptions/<subid>/resourceGroups/<rgname>/providers/Microsoft.Network/privateDnsZones/privatelink.${local.tags.region}.azmk8s.io"
  resource_group = {
    name     = var.rg_name
    location = var.tags.region
  }
  depends_on = [
    # data.azurerm_resource_group.example
    azurerm_resource_group.example
  ]
}

# data "azuread_group" "kube_admins" {
#   display_name     = "kube_admins"
#   security_enabled = true
# }
