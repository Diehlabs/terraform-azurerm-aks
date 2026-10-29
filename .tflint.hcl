config {
  call_module_type = "all"
  force = false
  disabled_by_default = false
}

plugin "azurerm" {
  enabled = true
  version = "0.33.0"
  source  = "github.com/terraform-linters/tflint-ruleset-azurerm"
}
