config {
  module = true
  force = false
  disabled_by_default = false
}

plugin "azurerm" {
  enabled = true
  version = "0.17.0"
  source  = "github.com/terraform-linters/tflint-ruleset-azurerm"
}
