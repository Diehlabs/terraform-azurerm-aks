#!/bin/sh

# exported items cause terratest to skip that stage
export SKIP_terraform_deploy=true
export SKIP_run_tests=true
export SKIP_terraform_redeploy=true
#export SKIP_terraform_destroy=true

# assuming you already ran `vault login -method=approle username=xxx`
approle_name="cloudauto_nonprod_ro"
# approle_name="k8sauto_nonprod_ro"
export VAULT_APPROLE_ID=$(vault read -format=json "auth/approle/role/${approle_name}/role-id" | jq -r .data.role_id)
export VAULT_WRAPPED_TOKEN=$(vault write -f -format=json -wrap-ttl=5m "auth/approle/role/${approle_name}/secret-id" | jq -r .wrap_info.token)
echo "Using approle ${approle_name}"
echo "Using approle ID ${VAULT_APPROLE_ID}"
echo "Using wrapped token ${VAULT_WRAPPED_TOKEN}"
go test -v -timeout 40m