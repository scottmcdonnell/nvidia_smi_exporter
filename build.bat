@echo off & setlocal
set batchPath=%~dp0
echo "building nvidia_smi_exporter.go..."
go build -v "%batchPath%\nvidia_smi_exporter.go"

echo "building installer/Output/nvidia_smi_exporter.msi..."
REM .\windows_installer\build.ps1 -PathToExecutable .\nvidia_smi_exporter.exe -Version "0.0.0" -Arch "amd64"
powershell.exe -noexit -file "%batchPath%\windows_installer\build.ps1" -PathToExecutable "%batchPath%\nvidia_smi_exporter.exe" --Version "0.0.0" -Arch "amd64"
