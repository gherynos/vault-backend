# Vault Backend

[![pre-commit](https://img.shields.io/badge/pre--commit-enabled-brightgreen?logo=pre-commit&logoColor=white)](https://github.com/pre-commit/pre-commit)
![build](https://github.com/gherynos/vault-backend/workflows/build/badge.svg)
![release](https://github.com/gherynos/vault-backend/workflows/release/badge.svg)
[![go-report-card](https://goreportcard.com/badge/github.com/gherynos/vault-backend)](https://goreportcard.com/report/github.com/gherynos/vault-backend)

A Terraform [HTTP backend](https://www.terraform.io/docs/backends/types/http.html) that stores the state in a [Vault secret](https://www.vaultproject.io/docs/secrets/kv/kv-v2).

The server supports locking and leverages the versioning capabilities of Vault by creating a new secret version when creating/updating the state.

## Terraform config

The server authenticates to Vault using [AppRole](https://www.vaultproject.io/docs/auth/approle), with `role_id` and `secret_id` passed respectively as the `username` and `password` in the configuration:

```terraform
terraform {
  backend "http" {
    address = "http://localhost:8080/state/<STATE_NAME>"
    lock_address = "http://localhost:8080/state/<STATE_NAME>"
    unlock_address = "http://localhost:8080/state/<STATE_NAME>"

    username = "<VAULT_ROLE_ID>"
    password = "<VAULT_SECRET_ID>"
  }
}
```

or directly with a [token](https://www.vaultproject.io/docs/auth/token):

```terraform
terraform {
  backend "http" {
    address = "http://localhost:8080/state/<STATE_NAME>"
    lock_address = "http://localhost:8080/state/<STATE_NAME>"
    unlock_address = "http://localhost:8080/state/<STATE_NAME>"

    username = "TOKEN"
    password = "<TOKEN_VALUE>"
  }
}
```

where `<STATE_NAME>` is an arbitrary value used to distinguish the backends.

With the above configuration, Terraform connects to a vault-backend server running locally on port 8080 when loading/storing/locking the state, and the server manages the following secrets in Vault:

- `/<VAULT_STORE>/<VAULT_PREFIX>/<STATE_NAME>`
- `/<VAULT_STORE>/<VAULT_PREFIX>/<STATE_NAME>-lock`

the latter gets created when a lock is acquired and deleted when released.

## Vault Backend config

The following environment variables can be set to change the configuration:

- `VAULT_URL` (default `http://localhost:8200`) the URL of the Vault server
- `VAULT_PREFIX` (default `vbk`) the prefix used when storing the secrets
- `VAULT_STORE` (default `secret`) the store path used when storing secrets
- `LISTEN_ADDRESS` (default `0.0.0.0:8080`) the listening address and port
- `TLS_CRT` and `TLS_KEY` to set the path of the TLS certificate and key files
- `DEBUG` to enable verbose logging

## Vault policy

The policy associated to the AppRole used by the server needs to grant access to the secrets.

I.e., for a `<STATE_NAME>` set as `cloud-services` and the default `VAULT_PREFIX` and `VAULT_STORE`:

```vault
path "secret/data/vbk/cloud-services"
{
  capabilities = ["create", "read", "update"]
}

path "secret/data/vbk/cloud-services-lock"
{
  capabilities = ["create", "read", "update"]
}

path "secret/metadata/vbk/cloud-services-lock"
{
  capabilities = ["delete"]
}
```

## Author

> GitHub [@gherynos](https://github.com/gherynos)

## License

Vault Backend is licensed under the [Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0).
