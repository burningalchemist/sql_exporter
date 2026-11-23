# AWS IAM Authentication

SQL Exporter can connect to Amazon RDS and Aurora (PostgreSQL, MySQL) with
[IAM database authentication](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.IAMDBAuth.html)
instead of a static password. It's supported for PostgreSQL and MySQL.

Query parameters for IAM authentication:

- `aws-iam-auth=true` to authenticate with an IAM auth token (**required**);
- `aws-iam-region=<AWS_REGION>` if SQL Exporter runs in a different region than the database, or environment has no region configured (optional).
- `aws-iam-role-arn=<ROLE_ARN>` to assume an IAM role and sign the token with its credentials, e.g. for a database in another account (optional).

The connection string would look like this:

```text
# Without TLS verification
postgres://user@mydb.abcdefgh.eu-west-1.rds.amazonaws.com:5432/dbname?sslmode=require&aws-iam-auth=true
mysql://user@mydb.abcdefgh.eu-west-1.rds.amazonaws.com:3306/dbname?tls=skip-verify&aws-iam-auth=true

# With TLS verification
postgres://user@mydb.abcdefgh.eu-west-1.rds.amazonaws.com:5432/dbname?sslmode=verify-full&sslrootcert=/path/to/global-bundle.pem&aws-iam-auth=true
mysql://user@mydb.abcdefgh.eu-west-1.rds.amazonaws.com:3306/dbname?tls=custom&tls-ca=/path/to/global-bundle.pem&aws-iam-auth=true
```

`sslmode=require` and `tls=skip-verify` encrypt the connection, but don't verify the server certificate.
`sslmode=verify-full` and `tls=custom` (see [MySQL Custom Certificates](mysql_tls.md)) also verify it with the
[RDS certificate bundle](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.SSL.html#UsingWithRDS.SSL.CertificatesAllRegions).

AWS credentials are taken from the default credential chain. The identity needs the `rds-db:connect` permission for
the database user. With `aws-iam-role-arn`, the identity needs `sts:AssumeRole` on the role instead, and the role needs
`rds-db:connect`. A token is valid for 15 minutes, so SQL Exporter generates a new one for every new database
connection.

A few things to keep in mind:

- Only `aws-iam-auth=true` enables IAM authentication. To disable it, remove the parameter instead of setting it to
  `false`.
- For PostgreSQL, the connection is always opened with the `pgx` driver, for every scheme (`postgres://`,
  `postgresql://`, `pg://`, `pgsql://`, `pgx://`). `lib/pq` is not used for IAM authentication.
- A password in the connection string is rejected, as the token is used as the password.
- For PostgreSQL, multiple hosts in the connection string (`host1,host2`) are rejected, as the token is signed for a
  single host.
- AWS IAM Authentication has some restrictions:
  - Host can't be an IP address or a custom DNS record.
  - TLS is required: see [`sslmode`](https://www.postgresql.org/docs/current/libpq-ssl.html#LIBPQ-SSL-PROTECTION) for
    PostgreSQL, [`tls`](https://github.com/go-sql-driver/mysql#tls) for MySQL. SQL Exporter rejects a connection
    string that can connect without TLS: `sslmode` has to be `require`, `verify-ca` or `verify-full` (not the default
    `prefer`), and `tls` has to be `true`, `skip-verify` or `custom` (not `preferred`).
  - For MySQL, the token is sent as a cleartext password, so `allowCleartextPasswords` is enabled automatically.
  - For MySQL, `tls=true` checks the server certificate against the system CA certificates, and the Amazon RDS CA is
    usually not one of them (`x509: certificate signed by unknown authority`). Use `tls=skip-verify`, or `tls=custom`
    with `tls-ca` (see [MySQL Custom Certificates](mysql_tls.md)) instead.
  - The database user has to be set up for IAM authentication:
    - Postgres: [`rds_iam`](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.IAMDBAuth.DBAccounts.html#UsingWithRDS.IAMDBAuth.DBAccounts.PostgreSQL)
    - MySQL: [`AWSAuthenticationPlugin`](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.IAMDBAuth.DBAccounts.html#UsingWithRDS.IAMDBAuth.DBAccounts.MySQL)

## Example setup

### PostgreSQL

1. Enable IAM database authentication on the DB instance (RDS) or on the DB cluster (Aurora).

2. Create the database user and `GRANT` it the `rds_iam` role:

   ```sql
   CREATE USER monitoring;
   GRANT rds_iam TO monitoring;
   ```

   > [!WARNING]
   > A user with the `rds_iam` role can no longer log in with a password, as
   > [IAM authentication takes precedence](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.IAMDBAuth.html#UsingWithRDS.IAMDBAuth.Limitations).
   > Don't grant it to the master user, or to a user that something else connects as with a password.

3. Give the necessary IAM policy to the role SQL Exporter uses (ECS task role, EC2 instance profile, etc.):

   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Action": "rds-db:connect",
         "Resource": "arn:aws:rds-db:eu-west-1:123456789012:dbuser:cluster-ABCDEFGHIJKL01234/monitoring"
       }
     ]
   }
   ```

   The resource ID in the ARN is the cluster resource ID (`cluster-...`) for Aurora, and the instance resource ID
   (`db-...`) for an RDS instance.

4. Use the connection string without a password:

   ```yaml
   target:
     data_source_name: 'postgres://monitoring@mydb.cluster-abcdefgh.eu-west-1.rds.amazonaws.com:5432/postgres?sslmode=require&aws-iam-auth=true'
   ```

5. To also verify the server certificate, download the
   [RDS certificate bundle](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.SSL.html#UsingWithRDS.SSL.CertificatesAllRegions)
   and set it in the connection string:

   ```shell
   curl -o /path/to/global-bundle.pem https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem
   ```

   ```yaml
   target:
     data_source_name: 'postgres://monitoring@mydb.cluster-abcdefgh.eu-west-1.rds.amazonaws.com:5432/postgres?sslmode=verify-full&sslrootcert=/path/to/global-bundle.pem&aws-iam-auth=true'
   ```

### MySQL

1. Enable IAM database authentication on the DB instance (RDS) or on the DB cluster (Aurora).

2. Create the database user with the `AWSAuthenticationPlugin`:

   ```sql
   CREATE USER 'monitoring'@'%' IDENTIFIED WITH AWSAuthenticationPlugin AS 'RDS';
   ```

   > [!WARNING]
   > A user created with the `AWSAuthenticationPlugin` can only log in with an IAM auth token, not with a password.
   > Don't change the master user, or a user that something else connects as with a password, to this plugin.

3. Give the necessary IAM policy to the role SQL Exporter uses (ECS task role, EC2 instance profile, etc.):

   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Action": "rds-db:connect",
         "Resource": "arn:aws:rds-db:eu-west-1:123456789012:dbuser:db-ABCDEFGHIJKL01234/monitoring"
       }
     ]
   }
   ```

   The resource ID in the ARN is the instance resource ID (`db-...`) for an RDS instance, and the cluster resource ID
   (`cluster-...`) for Aurora.

4. Use the connection string without a password:

   ```yaml
   target:
     data_source_name: 'mysql://monitoring@mydb.abcdefgh.eu-west-1.rds.amazonaws.com:3306/dbname?tls=skip-verify&aws-iam-auth=true'
   ```

5. To also verify the server certificate, download the
   [RDS certificate bundle](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.SSL.html#UsingWithRDS.SSL.CertificatesAllRegions)
   and set it in the connection string:

   ```shell
   curl -o /path/to/global-bundle.pem https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem
   ```

   ```yaml
   target:
     data_source_name: 'mysql://monitoring@mydb.abcdefgh.eu-west-1.rds.amazonaws.com:3306/dbname?tls=custom&tls-ca=/path/to/global-bundle.pem&aws-iam-auth=true'
   ```

   `tls=custom` and `tls-ca` are the parameters of [MySQL Custom Certificates](mysql_tls.md).
