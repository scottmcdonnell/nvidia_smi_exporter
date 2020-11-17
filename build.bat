@echo off & setlocal
set batchPath=%~dp0

set AppName=nvidia_smi_exporter
set DefaultPort=9202

REM get the version number
set Version=%1
if NOT [%~1]==[] goto :done
for /f "delims=" %%i in ('git describe --always') do set Version=%%i

:done

del "%batchPath%\bin\%AppName%.exe"
echo "APP: %AppName% Version: %Version%"
echo "building go application..."
echo go build -v -ldflags "-X main.version=%Version%" -o "bin/%AppName%.exe" . 
go build -v -ldflags "-X main.version=%Version%" -o "bin/%AppName%.exe" . 

echo "building installer/Output/%AppName%.msi..."
REM .\windows_installer\build.ps1 -PathToExecutable .\nvidia_smi_exporter.exe -Version "0.0.0" -Arch "amd64"

echo powershell.exe ^
-file "%batchPath%\windows_installer\build.ps1" ^
-BinDirectory "%batchPath%\bin" ^
-AppName "%AppName%" -DefaultPort "%DefaultPort%" ^
-Version "%Version%" -Arch "amd64" -Verbose

powershell.exe ^
-file "%batchPath%\windows_installer\build.ps1" ^
-BinDirectory "%batchPath%\bin" ^
-AppName "%AppName%" -DefaultPort "%DefaultPort%" ^
-Version "%Version%" -Arch "amd64" -Verbose
