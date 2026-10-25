# Secrets

Secret references are supported for both single-target and multi-target (jobs) setups, so you can use them in both
cases without any issues. Just make sure to use the correct syntax and provide the necessary parameters for the secret
manager you choose. Also check the permissions and access policies for the secret manager to ensure that SQL Exporter
has the necessary access to read the secrets.

Secrets are **only** resolved at startup, so if the secret value changes, you may need to restart SQL Exporter to pick
up the new value. Or use the `/reload` endpoint to trigger a configuration reload without restarting the process, but
keep in mind that this will also reload the entire configuration, not just the secrets.

SQL Exporter supports multiple secret management backends:

- Kubernetes Secrets
- AWS Secrets Manager
- Google Cloud Secret Manager
- Hashicorp Vault

## **Kubernetes Secrets**:

```text
k8ssecret://[namespace/]secret-name?key=field&template=dsn_template
```

Recommended for Kubernetes deployments. Requires RBAC permissions for the service account to read secrets. See
[k8s-secret example](https://github.com/burningalchemist/sql_exporter/examples/k8s-secret/) for detailed setup
instructions and the Helm chart automatically creates necessary RBAC resources.

## **AWS/GCP Secret Manager**:

If the database runs on AWS or Google Cloud, you might want to store the DSN in their Secret Manager services and allow
SQL Exporter to access it from there. This way you can avoid hardcoding credentials in the configuration file and
benefit from the security features of these services. In addition, Vault is also available as a secret manager option
for SQL Exporter.

The secrets can be referenced in the configuration file as a value for `data_source_name` item using the following
syntax:

```text
awssecretsmanager://<SECRET_NAME>?region=<AWS_REGION>&key=<JSON_KEY>
gcpsecretmanager://projects/<GCP_PROJECT_ID>/secrets/<SECRET_NAME>?key=<JSON_KEY>
```

The secret value can be a simple string or a JSON object:

- If it's a JSON object, you need to specify the `key` query parameter to indicate which value to use as the DSN;
- If the secret is a valid JSON but the `key` is not specified, SQL Exporter will try to use the value of
`data_source_name` key by default;
- If it's a simple string, then it will be used as a connection string directly. 

Using JSON format gives more flexibility and allows to store additional information or multiple DSNs in the same secret
resource.

## **Hashicorp Vault**

```text
hashivault://<MOUNT>/<SECRET_PATH>?key=<JSON_KEY>
```

The secret is read from a KV version 2 secrets engine by default. If the mount uses KV version 1, add `engine_version=1`
to the query parameters.

For Vault, you also need to specify the `VAULT_ADDR` and `VAULT_TOKEN` environment variables to let SQL Exporter
authenticate. This is a regular practice and goes beyond the scope of this document, so please refer to Vault
documentation for more details on how to set up and use Vault for secrets management.
