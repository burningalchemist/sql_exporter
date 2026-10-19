# Rate Limiting

To use rate limiting for incoming requests, you need to pass a configuration file using the `--web.config.file`
parameter. In the configuration file you need to specify rate limiting settings as in the example below:

```yaml
...
rate_limit:
  interval: "1s" # time interval between two requests, set to 0 to disable rate limiter
  burst: 20 # and permits a burst of up to 20 requests.
...
```

The format of the file is described in the
[exporter-toolkit](https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md) repository.
Also please check the available
[example](https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-config.yml).
