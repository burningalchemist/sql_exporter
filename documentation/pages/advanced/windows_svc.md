# Run as a Windows Service

If you run SQL Exporter from Windows, it might come in handy to register it as a service to avoid interactive sessions.
It is **important** to define `--config.file` parameter to load the configuration file. The other settings can be added
as well. The registration itself is performed with Powershell or CMD (make sure you run it as `Administrator`):

```powershell
New-Service -name "SqlExporterSvc" `
-BinaryPathName "%SQL_EXPORTER_PATH%\sql_exporter.exe --config.file %SQL_EXPORTER_PATH%\sql_exporter.yml" `
-StartupType Automatic `
-DisplayName "Prometheus SQL Exporter"
```

```cmd
sc.exe create SqlExporterSvc binPath= "%SQL_EXPORTER_PATH%\sql_exporter.exe --config.file %SQL_EXPORTER_PATH%\sql_exporter.yml" start= auto
```

`%SQL_EXPORTER_PATH%` is a path to the SQL Exporter binary executable. This document assumes that configuration files
are in the same location.

In case you need a more sophisticated setup (e.g. with logging, environment variables, etc), you might want to use
[NSSM](https://nssm.cc/) or [WinSW](https://github.com/winsw/winsw). Please consult their documentation for more
details.
