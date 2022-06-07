package test

import (
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/azure"
	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	roleID          = os.Getenv("VAULT_APPROLE_ID")
	wrappedToken    = os.Getenv("VAULT_WRAPPED_TOKEN")
	vaultSecretPath = "cloudauto/data/terraform/nonprod/azure/svcazsp-cloudauto-terratest-devtest-terraform-managed"

	uniqueId           = random.UniqueId()
	terraformBinary    = "/usr/local/bin/terraform"
	workingDir         = "../examples/build"
	expectedAgentCount = 2
)

// We want to set a bunch of env vars, some from vault some from static values:

// These are env var name : env var value from vault
var vaultSecretMap = map[string]string{
	"ARM_CLIENT_ID":         "client_id",
	"ARM_CLIENT_SECRET":     "client_secret",
	"ARM_TENANT_ID":         "tenant_id",
	"ARM_SUBSCRIPTION_ID":   "subscription_id",
	"AZURE_CLIENT_ID":       "client_id",
	"AZURE_CLIENT_SECRET":   "client_secret",
	"AZURE_TENANT_ID":       "tenant_id",
	"AZURE_SUBSCRIPTION_ID": "subscription_id",
}

// These are env var name : env var value, as-is
var goEnvVars = map[string]string{
	"ARM_AKS_KUBE_CONFIGS_SENSITIVE": "true",
}

func TestPrivateAksModule(t *testing.T) {
	t.Parallel()

	// os.Setenv("SKIP_reterraform_deploy", "true")
	// os.Setenv("SKIP_terraform_redeploy", "true")
	// os.Setenv("SKIP_terraform_destroy", "true")

	if tfcp := os.Getenv("AGENT_TEMPDIRECTORY"); tfcp != "" {
		terraformBinary = tfcp + "/terraform"
	}

	if tfdir := os.Getenv("TERRATEST_WORKING_DIR"); tfdir != "" {
		workingDir = tfdir
	}

	if unid := os.Getenv("BUILD_BUILDID"); unid != "" {
		uniqueId = unid
	}

	// func SaveString(t testing.TestingT, testFolder string, name string, val string) {
	// 	path := formatNamedTestDataPath(testFolder, name)
	// 	SaveTestData(t, path, val)
	// }

	terraformVars := map[string]interface{}{
		"unique_id":  uniqueId,
		"rg_name":    fmt.Sprintf("aks-terratest-%s", uniqueId), //fmt.Sprintf("terratest-%s", uniqueId),
		"node_count": expectedAgentCount,
		"msi_id":     "/subscriptions/<subid>/resourceGroups/<rgname>/providers/Microsoft.ManagedIdentity/userAssignedIdentities/<identityname>",
		"tags": map[string]string{
			"cost_center":       "01245",
			"environment":       "nonprod",
			"owner":             "Chris Diehl",
			"technical_contact": "devops@diehlabs.com",
			"product":           "terratest",
			"region":            "westus",
		},
	}

	terraformEnvVars := map[string]string{}

	// terraformOptions := setupTesting(t, workingDir, terraformBinary, terraformVars, roleID, wrappedToken, vaultSecretPath, vaultSecretMap)

	// return terraformOptions if using cluster tests
	// terraformOptions := SetupTesting(t, workingDir, terraformBinary, terraformVars, terraformEnvVars)
	// don't return if not using cluster tests
	SetupTesting(t, workingDir, terraformBinary, terraformVars, terraformEnvVars)

	SetAzureEnvVars(t, roleID, wrappedToken, vaultSecretPath, vaultSecretMap)

	// get secrets
	// getSpn, err := terratest_helper.GetSecretWithAppRole(roleId, secretId, vaultSecretPath)
	// if err != nil {
	// 	fmt.Print(err)
	// 	panic(err)
	// }
	// fmt.Sprintf("Using SPN: %s", getSpn)

	// Destroy the infra after testing is finished
	defer test_structure.RunTestStage(t, "terraform_destroy", func() {
		TerraformDestroy(t, workingDir)
	})

	// Deploy using Terraform
	test_structure.RunTestStage(t, "terraform_deploy", func() {
		DeployUsingTerraform(t, workingDir)
	})

	// the module generates the cluster name, so we get the expected name from the outputs of the test configuration
	// expectedClusterName := terraform.Output(t, terraformOptions, "cluster_name")
	// expectedResourceGroupName := terraformOptions.Vars["rg_name"].(string)

	// run tests
	// test_structure.RunTestStage(t, "run_tests", func() {
	// 	t.Run("Misc k8s cluster tests", func(t *testing.T) {
	// 		testCluster(t, workingDir, expectedResourceGroupName, expectedClusterName)
	// 	})
	// })

	// t.Run("Misc k8s cluster tests", func(t *testing.T) {
	// 	testCluster(t, workingDir, expectedResourceGroupName, expectedClusterName)
	// })
	// test_structure.RunTestStage(t, "cluster_tests", func() {
	// 	testCluster(t, workingDir, expectedResourceGroupName, expectedClusterName)
	// })

	// Redeploy using Terraform and ensure idempotency
	test_structure.RunTestStage(t, "terraform_redeploy", func() {
		RedeployUsingTerraform(t, workingDir)
	})

}

func deployIngress(t *testing.T) {
	kubeResourcePath, err := filepath.Abs(workingDir + "/deployments/nginx-ingress.yaml")
	require.NoError(t, err)

	options := k8s.NewKubectlOptions("", fmt.Sprintf("%s/kubeconfig", workingDir), "ingress-nginx")

	// This will run `kubectl apply -f RESOURCE_CONFIG` and fail the test if there are any errors
	k8s.KubectlApply(t, options, kubeResourcePath)
}

// to skip execution "export SKIP_test_resources=true" in terminal
func testCluster(t *testing.T, workingDir string, resourceGroupName string, expectedClusterName string) {
	// Look up the cluster node count
	cluster, err := azure.GetManagedClusterE(t, resourceGroupName, expectedClusterName, "")
	require.NoError(t, err)
	actualCount := *(*cluster.ManagedClusterProperties.AgentPoolProfiles)[0].Count

	// Test that the Node count matches the Terraform specification
	assert.Equal(t, int32(expectedAgentCount), actualCount)

	// deploy nginx ingress controller for testing
	deployIngress(t)

	// Path to the Kubernetes resource config we will test
	kubeResourcePath, err := filepath.Abs(workingDir + "/deployments/nginx-deployment.yaml")
	require.NoError(t, err)

	// To ensure we can reuse the resource config on the same cluster to test different scenarios, we setup a unique
	// namespace for the resources for this test.
	// Note that namespaces must be lowercase.
	namespaceName := strings.ToLower(random.UniqueId())

	// Setup the kubectl config and context. Here we choose to use the defaults, which is:
	// - HOME/.kube/config for the kubectl config file
	// - Current context of the kubectl config file
	options := k8s.NewKubectlOptions("", fmt.Sprintf("%s/kubeconfig", workingDir), namespaceName)

	k8s.CreateNamespace(t, options, namespaceName)
	// ... and make sure to delete the namespace at the end of the test
	defer k8s.DeleteNamespace(t, options, namespaceName)

	// At the end of the test, run `kubectl delete -f RESOURCE_CONFIG` to clean up any resources that were created.
	defer k8s.KubectlDelete(t, options, kubeResourcePath)

	//
	// generate a csr + key
	// have the cluster sign the csr
	// store the cert + key in k8s secret
	// ensure the deployment load balancer service can access the secret
	// serviceCert := generateCsr(expectedClusterName)

	// This will run `kubectl apply -f RESOURCE_CONFIG` and fail the test if there are any errors
	k8s.KubectlApply(t, options, kubeResourcePath)

	// This will wait up to 10 seconds for the service to become available, to ensure that we can access it.
	k8s.WaitUntilServiceAvailable(t, options, "nginx-service", 10, 20*time.Second)
	// Now we verify that the service will successfully boot and start serving requests
	service := k8s.GetService(t, options, "nginx-service")
	endpoint := k8s.GetServiceEndpoint(t, options, service, 443)

	// Setup a TLS configuration to submit with the helper, a blank struct is acceptable
	tlsConfig := tls.Config{}

	// Test the endpoint for up to 5 minutes. This will only fail if we timeout waiting for the service to return a 200
	// response.
	http_helper.HttpGetWithRetryWithCustomValidation(
		t,
		fmt.Sprintf("http://%s", endpoint),
		&tlsConfig,
		30,
		10*time.Second,
		func(statusCode int, body string) bool {
			return statusCode == 200
		},
	)
}
