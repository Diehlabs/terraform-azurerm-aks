provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "terratest" {
  name     = var.rg_name
  location = local.tags.region
  tags     = local.tags
}

resource "tls_private_key" "example" {
  algorithm = "RSA"
  rsa_bits  = 2048
}

resource "azurerm_virtual_network" "terratest" {
  name                = "vnet-terratest-${var.unique_id}"
  location            = azurerm_resource_group.terratest.location
  resource_group_name = azurerm_resource_group.terratest.name
  address_space       = ["172.16.14.0/24"]
  tags                = local.tags
}

resource "azurerm_subnet" "terratest" {
  name                 = "subnet-terratest-${var.unique_id}"
  resource_group_name  = azurerm_resource_group.terratest.name
  virtual_network_name = azurerm_virtual_network.terratest.name
  address_prefixes     = ["172.16.14.0/24"]
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
  resource_group_name       = var.rg_name
  tags                      = local.tags
  subnet_id                 = azurerm_subnet.terratest.id
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
  # msi_id            = var.msi_id #data.terraform_remote_state.devtest_infra.outputs.identity.id
  cluster_admin_ids = ["9ba4a348-227d-4411-bc37-3fb81ee8bc48"]
  # laws                = data.azurerm_log_analytics_workspace.example
}

# data "azuread_group" "kube_admins" {
#   display_name     = "kube_admins"
#   security_enabled = true
# }
