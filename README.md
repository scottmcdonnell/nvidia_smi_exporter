# nvidia_smi_exporter

A Prometheus exporter for Nvidia metrics.
This exporter used the [nvidia-smi](https://developer.nvidia.com/nvidia-system-management-interface) command line tool

This project is a mixture of [phstudy/nvidia_smi_exporter](https://github.com/phstudy/nvidia_smi_exporter) and [zhebrak/nvidia_smi_exporter](https://github.com/zhebrak/nvidia_smi_exporter) with a windows service added. 

## Todos
 - Crossplatform so that Windows only service elements are only built on Windows.
 - Tests.

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

| Flag | Description | Default Value 
|------|-------------|--------------
| `telemetry.addr`   | host:port for exporter.                 | `:9202` 
| `--telemetry.path` | URL Path under which to expose metrics. | `/metrics` 
| `--help`           | Show context-sensitive help.            |           
| `--version`        | Show application version.               |    

# Service

Build the service for Windows:

    build.bat

or specify a version number:

    build.bat 0.0.0

This creates an .msi installer in the folder `bin`

The installer will setup the nvidia_smi_exporter as a Windows service, as well as create an exception in the Windows Firewall.

If the installer is run without any parameters, the exporter will run with default settings for enabled ports, etc. The following parameters are available:

Name | Description
-----|------------
`LISTEN_ADDR` | The IP address to bind to. Defaults to 0.0.0.0
`LISTEN_PORT` | The port to bind to. Defaults to 9182.
`METRICS_PATH` | The path at which to serve metrics. Defaults to `/metrics`
`REMOTE_ADDR` | Allows setting comma separated remote IP addresses for the Windows Firewall exception (whitelist). Defaults to an empty string (any remote address).
`COMMAND_NAME` | Name of command to execute. Defaults to `nvidia-smi`
`COMMAND_FLAGS` | Flags for command to execute. Defaults to `-q -x` for query in XML.

Parameters are sent to the installer via `msiexec`. Example invocations:

```powershell
msiexec /i <path-to-msi-file> LISTEN_PORT=5000
```

On some older versions of Windows you may need to surround parameter values with double quotes to get the install command parsing properly:
```powershell
msiexec /i <path-to-msi-file> LISTEN_PORT="5000"
```

The nvidia-smi executable is looked for in the following locations:

    "C:\\Program Files\\NVIDIA Corporation\\NVSMI\\nvidia-smi.exe"
    "C:\\Windows\\System32\\nvidia-smi.exe"
    "/usr/bin/nvidia-smi"

If the nvidia-smi executable is at a different location then use the COMMAND_NAME flag below to send the full path to the location eg. 

```powershell
msiexec /i nvidia_smi_exporter-0.0.4-amd64.msi COMMAND_NAME="C:\Windows\System32\nvidia-smi.exe"
```


for debugging installer issues try:

    msiexec /i nvidia_smi_exporter-0.0.2-amd64.msi /L*V "package.log"

### Prometheus example config

```
- job_name: "nvidia_exporter"
  static_configs:
  - targets: ['localhost:9202']
```

# Boilerplate

This project serves as a boilerplate for a commandline prometheus exporter.

To implement a new exporter `main.go` can stay as is. `metrics.go` can be updated with the new constants and metrics.

Installer build can be updated by editing the variables in `build.bat`

## GUID
A new GUID will be needed in the `windows_installer\exporter.wxs` for the `UpgradeCode` - this unique id identifies the project so the version can be checked on install. 

    <Product Id="*" UpgradeCode="702e1894-0110-4f1e-80c1-8e587e4cb51e"
           Name="$(var.AppName)" Version="$(var.Version)" Manufacturer="my_company"
           Language="1033" Codepage="1252">

A new GUID can be created for a new project using [online-guid-generator](https://www.guidgenerator.com/online-guid-generator.aspx)
