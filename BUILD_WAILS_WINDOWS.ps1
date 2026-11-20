#requires -Version 5.1
$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath $PSScriptRoot

# Windows PowerShell 5.1 needs a UTF-8 BOM to decode this file's Chinese text.
try {
    [Console]::OutputEncoding = New-Object System.Text.UTF8Encoding($false)
    $OutputEncoding = New-Object System.Text.UTF8Encoding($false)
} catch {}

$exitCode = 0
$logDir = Join-Path $PSScriptRoot 'build\logs'
New-Item -ItemType Directory -Force -Path $logDir | Out-Null
$logFile = Join-Path $logDir ( 'wails-build-' + (Get-Date -Format 'yyyyMMdd-HHmmss') + '.log' )
$transcriptStarted = $false

function Invoke-Native {
    param(
        [Parameter(Mandatory=$true)][string]$Executable,
        [Parameter(Mandatory=$true)][string[]]$Arguments,
        [Parameter(Mandatory=$true)][string]$Step
    )
    Write-Host "`n[$Step] $Executable $($Arguments -join ' ')" -ForegroundColor Cyan
    & $Executable @Arguments
    $nativeCode = $LASTEXITCODE
    if ($nativeCode -ne 0) {
        throw "$Step 失败，退出代码：$nativeCode。请查看完整错误和日志：$logFile"
    }
}

try {
    Start-Transcript -Path $logFile -Force | Out-Null
    $transcriptStarted = $true

    Write-Host 'ORFix Repair V4.1.1 | Wails 原生 Windows 桌面版构建' -ForegroundColor Cyan
    Write-Host "工程目录：$PSScriptRoot"
    Write-Host "日志文件：$logFile"

    $goCommand = Get-Command go.exe -ErrorAction Stop
    $goExe = $goCommand.Source
    $versionText = (& $goExe version 2>&1 | Out-String).Trim()
    if ($LASTEXITCODE -ne 0) { throw "执行 go version 失败：$versionText" }
    Write-Host "Go：$versionText" -ForegroundColor Green

    $match = [regex]::Match($versionText, 'go(?<major>\d+)\.(?<minor>\d+)')
    if (-not $match.Success) { throw "无法解析 Go 版本：$versionText" }
    $major = [int]$match.Groups['major'].Value
    $minor = [int]$match.Groups['minor'].Value
    if (($major -lt 1) -or ($major -eq 1 -and $minor -lt 26)) {
        throw '当前工程使用 Wails v2.17.1，要求 Go 1.26 或更高版本。你当前的 Go 版本不足。'
    }

    # Go 1.26.4 is already installed, so do not let Go silently download another toolchain.
    $env:GOTOOLCHAIN = 'local'
    $proxyText = (& $goExe env GOPROXY 2>&1 | Out-String).Trim()
    $gopathText = (& $goExe env GOPATH 2>&1 | Out-String).Trim()
    if ($LASTEXITCODE -ne 0) { throw '读取 Go 环境变量失败。' }
    Write-Host "GOPROXY：$proxyText"
    Write-Host "GOPATH：$gopathText"

    Invoke-Native -Executable $goExe -Arguments @('mod','download') -Step '1/4 下载 Go/Wails 依赖'
    Invoke-Native -Executable $goExe -Arguments @('test','-count=1','./engine') -Step '2/4 测试 Mod 修复引擎'
    Invoke-Native -Executable $goExe -Arguments @('install','github.com/wailsapp/wails/v2/cmd/wails@v2.17.1') -Step '3/4 安装锁定版本的 Wails CLI'

    $gopath = (& $goExe env GOPATH 2>&1 | Out-String).Trim()
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($gopath)) {
        throw '无法读取 GOPATH。请在命令行执行：go env GOPATH'
    }
    $wailsExe = Join-Path $gopath 'bin\wails.exe'
    if (-not (Test-Path -LiteralPath $wailsExe)) {
        throw "Wails CLI 安装后仍未找到：$wailsExe"
    }

    Invoke-Native -Executable $wailsExe -Arguments @('build','-platform','windows/amd64','-s','-clean') -Step '4/4 构建 Wails 原生桌面 EXE'

    $exe = Join-Path $PSScriptRoot 'build\bin\ORFixRepair_v4_1_1.exe'
    if (-not (Test-Path -LiteralPath $exe)) {
        throw "Wails 命令结束但未找到预期文件：$exe。请检查 build\bin 目录和日志。"
    }

    $hash = (Get-FileHash -LiteralPath $exe -Algorithm SHA256).Hash
    Write-Host "`n构建成功：$exe" -ForegroundColor Green
    Write-Host "文件大小：$([math]::Round((Get-Item -LiteralPath $exe).Length / 1MB, 2)) MB"
    Write-Host "SHA256：$hash"
    Write-Host '这是 Wails 原生桌面窗口，不会启动 Edge 浏览器标签页。运行时需要 Microsoft Edge WebView2 Runtime。'
}
catch {
    $exitCode = 1
    Write-Host "`n构建失败：$($_.Exception.Message)" -ForegroundColor Red
    Write-Host '请不要关闭窗口。把上方从第一个错误开始的完整信息发给我。' -ForegroundColor Yellow
}
finally {
    if ($transcriptStarted) {
        try { Stop-Transcript | Out-Null } catch {}
    }
    Write-Host "`n构建日志：$logFile" -ForegroundColor Yellow
    if ($exitCode -eq 0) {
        Read-Host '构建完成。按 Enter 关闭窗口' | Out-Null
    } else {
        Read-Host '构建失败。按 Enter 关闭窗口（错误日志已保存）' | Out-Null
    }
}
exit $exitCode
