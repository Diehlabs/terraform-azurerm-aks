package test

import (
	"fmt"
	"os"
	"testing"

	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

func SetupTesting(
	t *testing.T,
	workingDir string,
	terraformBinary string,
	terraformVars map[string]interface{},
	terraformEnvVars map[string]string,
) *terraform.Options {

	testDataExists := test_structure.IsTestDataPresent(t, test_structure.FormatTestDataPath(workingDir, "TerraformOptions.json"))

	if testDataExists {
		logger.Logf(t, "Found and loaded test data in %s", workingDir)
		return test_structure.LoadTerraformOptions(t, workingDir)
	} else {
		terraformOptions := &terraform.Options{
			TerraformDir:    workingDir,
			TerraformBinary: terraformBinary,
			Vars:            terraformVars,
			EnvVars:         terraformEnvVars,
		}

		test_structure.SaveTerraformOptions(t, workingDir, terraformOptions)

		logger.Logf(t, "Saved test data in %s so it can be reused later", workingDir)

		return terraformOptions
	}
}

func DeployUsingTerraform(t *testing.T, workingDir string) {
	terraformOptions := test_structure.LoadTerraformOptions(t, workingDir)
	terraform.InitAndApply(t, terraformOptions)
}

func RedeployUsingTerraform(t *testing.T, workingDir string) {
	terraformOptions := test_structure.LoadTerraformOptions(t, workingDir)
	terraform.ApplyAndIdempotent(t, terraformOptions)
}

func TerraformDestroy(t *testing.T, workingDir string) {
	terraformOptions := test_structure.LoadTerraformOptions(t, workingDir)
	terraform.Destroy(t, terraformOptions)
	test_structure.CleanupTestDataFolder(t, workingDir)
}

func GetResourceTags(product string, region string, environment string, owner string, technicalContact string, costCenter string) map[string]string {
	resourceTags := map[string]string{
		"product":           product,
		"region":            region,
		"environment":       environment,
		"owner":             owner,
		"technical_contact": technicalContact,
		"cost_center":       costCenter,
	}
	return resourceTags
}

func GetSpnEnvVars(id string, secret string, sub string, tenant string) map[string]string {
	spnEnvVars := map[string]string{
		"ARM_CLIENT_ID":         id,
		"ARM_CLIENT_SECRET":     secret,
		"ARM_TENANT_ID":         tenant,
		"ARM_SUBSCRIPTION_ID":   sub,
		"AZURE_CLIENT_ID":       id,
		"AZURE_CLIENT_SECRET":   secret,
		"AZURE_TENANT_ID":       tenant,
		"AZURE_SUBSCRIPTION_ID": sub,
	}
	return spnEnvVars
}

func SetAzureEnvVars(t *testing.T, roleID string, wrappedToken string, vaultSecretPath string, vaultSecretMap map[string]string) {
	vaultSecretsMap := GetSecretWithAppRole(roleID, wrappedToken, vaultSecretPath, vaultSecretMap)
	for name, value := range vaultSecretsMap {
		fmt.Printf("Setting var: %s", name)
		os.Setenv(name, value)
	}
	// t.Logf("Terraform env vars are set.")
}
