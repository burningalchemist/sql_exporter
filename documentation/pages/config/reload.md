# Reload

To update the configuration of the exporter without restarting it, you may want to use the reload feature. This allows
to apply changes to the configuration file dynamically, discovering new collectors or updating existing ones without
downtime.

To enable reloading we have to pass the `--web.enable-reload` flag to the exporter command line. This will enable the
reload feature and expose an `/reload` HTTP endpoint or let send a SIGHUP signal to the exporter process to trigger a
reload.

What's allowed to be reloaded:

- global settings;
- collector settings (e.g. metrics, queries, adding/removing collectors);

What's not allowed to be reloaded and requires a restart:

- targets (connection strings);
- switching modes (`target` vs. `jobs`);

It is important to note that connections will not be changed upon the restart of the exporter, so if you change the
connection settings in the configuration file, you will need to restart the exporter for those changes to take effect.
This is intended behavior to avoid unnecessary connection churn and to maintain the state of the monitored services.
Security aspect is another reason as we don't want hijacked connections that might end up contributing to a
(Distributed) Denial of Service or any other malicious activity.
