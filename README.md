# nvidia_smi_exporter

A Prometheus exporter for Nvidia metrics.
This exporter used the [nvidia-smi](https://developer.nvidia.com/nvidia-system-management-interface) command line tool

This project is a mixture of [phstudy/nvidia_smi_exporter](https://github.com/phstudy/nvidia_smi_exporter) and [zhebrak/nvidia_smi_exporter](https://github.com/zhebrak/nvidia_smi_exporter) with a windows service added. 


# Building and Running
Prerequisites:

* [Go compiler](https://golang.org/dl/)

Building:

    go get github.com/scottmcdonnell/nvidia_smi_exporter
    cd ${GOPATH-$HOME/go}/src/github.com/scottmcdonnell/nvidia_smi_exporter
    go build -v -o bin/nvidia_smi_exporter -ldflags "-X main.version=1.0.0" .

Run with default port:

    bin/nvidia_smi_exporter

Run with specified port:

    bin/nvidia_smi_exporter --telemetry.addr ":9201"

Default port is 9201

# Application Flags
Exporter accepts flags to configure certain behaviours. The ones configuring the global behaviour of the exporter are listed below.

| Flag               | Description                             | Default Value |
|--------------------|-----------------------------------------|---------------
|
| `telemetry.addr`   | host:port for exporter.                 | `:9202`       |
| `--telemetry.path` | URL Path under which to expose metrics. | `/metrics`    |
| `--help`           | Show context-sensitive help.            |               |
| `--version`        | Show application version.               |    
|           

# Service

Build the service for Windows:

    build.bat

This creates an .msi installer in the folder `windows_installer/Output`

The installer will setup the nvidia_smi_exporter as a Windows service, as well as create an exception in the Windows Firewall.

If the installer is run without any parameters, the exporter will run with default settings for enabled ports, etc. The following parameters are available:

Name | Description
-----|------------
`LISTEN_ADDR` | The IP address to bind to. Defaults to 0.0.0.0
`LISTEN_PORT` | The port to bind to. Defaults to 9182.
`METRICS_PATH` | The path at which to serve metrics. Defaults to `/metrics`
`REMOTE_ADDR` | Allows setting comma separated remote IP addresses for the Windows Firewall exception (whitelist). Defaults to an empty string (any remote address).

Parameters are sent to the installer via `msiexec`. Example invocations:

```powershell
msiexec /i <path-to-msi-file> LISTEN_PORT=5000
```

On some older versions of Windows you may need to surround parameter values with double quotes to get the install command parsing properly:
```powershell
msiexec /i <path-to-msi-file> LISTEN_PORT="5000"
```

### Prometheus example config

```
- job_name: "nvidia_exporter"
  static_configs:
  - targets: ['localhost:9202']
```


Create a GUID
https://www.guidgenerator.com/online-guid-generator.aspx
