# MySQL Custom Certificates

SQL Exporter supports custom TLS certificates for MySQL connections. This is useful when you have a self-signed
certificate, a certificate from a private CA, or want to use mTLS for MySQL connections.

To use custom TLS certificates for MySQL connections, you need to add the following parameters to the connection string:

1. `tls=custom` to indicate that you want to use a custom TLS configuration (required to enable custom TLS support);
2. `tls-ca=<PATH_TO_CA_CERT>` to specify the path to the CA certificate file (if your certificate is self-signed or
   from a private CA);
3. `tls-cert=<PATH_TO_CLIENT_CERT>` and `tls-key=<PATH_TO_CLIENT_KEY>` to specify the paths to the client certificate
   and key files if you want to use mTLS (if your MySQL server requires client authentication).

The connection string would look like this:

```text
mysql://user:password@hostname:port/dbname?tls=custom&tls-ca=/path/to/ca.pem
mysql://user:password@hostname:port/dbname?tls=custom&tls-cert=/path/to/client-cert.pem&tls-key=/path/to/client-key.pem
```

This configuration is only applied to MySQL as there is no way to provide the configuration natively. For other
databases, you may want to consult their documentation on how to set up TLS connections and apply the necessary
parameters to the connection string if it's supported by the driver.

TLS Configuration is bound to the `hostname+port` combinations, so if there are connections using the same
`hostname+port` combination, they will share and re-use the same TLS configuration.
