package test

import (
	"context"
	"fmt"

	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/approle"
)

//------------------------------------------------------------------------------------------------
// Fetches a key-value secret (kv-v2) after authenticating via AppRole.
//------------------------------------------------------------------------------------------------
func GetSecretWithAppRole(
	roleID string,
	wrappedToken string,
	vaultSecretPath string,
	vaultSecretMap map[string]string,
) map[string]string {
	config := vault.DefaultConfig() // modify for more granular configuration

	client, err := vault.NewClient(config)
	if err != nil {
		panic(fmt.Errorf("unable to initialize Vault client: %w", err))
	}

	//------------------------------------------------------------------------------------------------
	// The Secret ID is a value that needs to be protected, so instead of the
	// app having knowledge of the secret ID directly, we have a trusted orchestrator (https://learn.hashicorp.com/tutorials/vault/secure-introduction?in=vault/app-integration#trusted-orchestrator)
	// give the app access to a short-lived response-wrapping token (https://www.vaultproject.io/docs/concepts/response-wrapping).
	// Read more at: https://learn.hashicorp.com/tutorials/vault/approle-best-practices?in=vault/auth-methods#secretid-delivery-best-practices
	//------------------------------------------------------------------------------------------------
	secretID := &auth.SecretID{FromString: wrappedToken}

	appRoleAuth, err := auth.NewAppRoleAuth(
		roleID,
		secretID,
		auth.WithWrappingToken(),
	)
	if err != nil {
		panic(fmt.Errorf("unable to initialize AppRole auth method: %w", err))
	}

	authInfo, err := client.Auth().Login(context.TODO(), appRoleAuth)
	if err != nil {
		panic(fmt.Errorf("unable to login to AppRole auth method: %w", err))
	}
	if authInfo == nil {
		panic(fmt.Errorf("no auth info was returned after login"))
	}

	// get secret
	secret, err := client.Logical().Read(vaultSecretPath)
	if err != nil {
		panic(fmt.Errorf("unable to read secret: %w", err))
	}

	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		panic(fmt.Errorf("data type assertion failed: %T %#v", secret.Data["data"], secret.Data["data"]))
	}

	terraformEnvVars, err := getTfEnvVars(data, vaultSecretMap)
	if err != nil {
		panic(fmt.Errorf("Cant get data"))
	} else {
		return terraformEnvVars
	}

}

func getTfEnvVars(data map[string]interface{}, vaultSecretMap map[string]string) (map[string]string, error) {
	for varName, secretName := range vaultSecretMap {
		value, ok := data[secretName].(string)
		if !ok {
			return map[string]string{}, fmt.Errorf("value type assertion failed: %T %#v", data[secretName], data[secretName])
		} else {
			vaultSecretMap[varName] = value
		}
	}

	return vaultSecretMap, nil
}
