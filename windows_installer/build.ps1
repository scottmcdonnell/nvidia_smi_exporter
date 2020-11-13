[CmdletBinding()]
Param (
    [Parameter(Mandatory = $true)]
    [String] $AppName,
    [Parameter(Mandatory = $true)]
    [String] $BinDirectory,
    [Parameter(Mandatory = $true)]
    [String] $DefaultPort,
    [Parameter(Mandatory = $true)]
    [String] $Version,
    [Parameter(Mandatory = $false)]
    [ValidateSet("amd64", "386")]
    [String] $Arch = "amd64"
)
$ErrorActionPreference = "Stop"

# Get absolute path to executable before switching directories
$PathToExecutable = Resolve-Path "${BinDirectory}\${AppName}.exe"
# Set working dir to this directory, reset previous on exit
Push-Location $PSScriptRoot
Trap {
    # Reset working dir on error
    Pop-Location
}

if ($PSVersionTable.PSVersion.Major -lt 5) {
    Write-Error "Powershell version 5 required"
    exit 1
}

$wc = New-Object System.Net.WebClient
function Get-FileIfNotExists {
    Param (
        $Url,
        $Destination
    )
    if (-not (Test-Path $Destination)) {
        Write-Verbose "Downloading $Url"
        $wc.DownloadFile($Url, $Destination)
    }
    else {
        Write-Verbose "${Destination} already exists. Skipping."
    }
}

$sourceDir = mkdir -Force Source
mkdir -Force Work | Out-Null

Write-Verbose "Downloading WiX..."
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
Get-FileIfNotExists "https://github.com/wixtoolset/wix3/releases/download/wix311rtm/wix311-binaries.zip" "$sourceDir\wix-binaries.zip"
mkdir -Force WiX | Out-Null
Expand-Archive -Path "${sourceDir}\wix-binaries.zip" -DestinationPath WiX -Force

Copy-Item -Force $PathToExecutable Work/${AppName}.exe

Write-Verbose "Creating ${AppName}-${Version}-${Arch}.msi"
$wixArch = @{"amd64" = "x64"; "386" = "x86"}[$Arch]
$wixOpts = "-ext WixFirewallExtension -ext WixUtilExtension"
Invoke-Expression "WiX\candle.exe -nologo -arch $wixArch $wixOpts -out Work\${AppName}.wixobj -dAppName=`"$AppName`"  -dVersion=`"$Version`" -dDefaultPort=`"$DefaultPort`" exporter.wxs"
Invoke-Expression "WiX\light.exe -nologo -spdb $wixOpts -out `"${BinDirectory}\${AppName}-${Version}-${Arch}.msi`" Work\${AppName}.wixobj"

Write-Verbose "Done!"
Pop-Location
