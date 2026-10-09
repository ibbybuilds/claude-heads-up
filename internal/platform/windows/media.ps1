# Pauses or resumes media through the Windows media session API (the same one
# behind the volume flyout). Every browser tab and app that shows up there is
# reachable, with no per-browser code.
#
# Input:  $env:HEADS_UP_ACTION = 'pause' | 'resume'
#         $env:HEADS_UP_MEDIA  = JSON array of {app, title} to resume
# Output: JSON array of {app, title} that were actually paused or resumed

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
[Console]::OutputEncoding = [Text.Encoding]::UTF8
Add-Type -AssemblyName System.Runtime.WindowsRuntime

# PowerShell cannot await WinRT IAsyncOperation directly; bridge through AsTask.
$asTask = [System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object {
    $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and
    $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation`1'
} | Select-Object -First 1

function Await($operation, [Type]$resultType) {
    $task = $asTask.MakeGenericMethod($resultType).Invoke($null, @($operation))
    $task.Wait(-1) | Out-Null
    $task.Result
}

$managerType = [Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager, Windows.Media.Control, ContentType = WindowsRuntime]
$propsType = [Windows.Media.Control.GlobalSystemMediaTransportControlsSessionMediaProperties, Windows.Media.Control, ContentType = WindowsRuntime]
$manager = Await ($managerType::RequestAsync()) $managerType

function Get-Title($session) {
    $props = Await ($session.TryGetMediaPropertiesAsync()) $propsType
    if ($props.Title) { $props.Title } else { '' }
}

$targets = @(ConvertFrom-Json $env:HEADS_UP_MEDIA)
$done = @()
foreach ($session in $manager.GetSessions()) {
    $status = $session.GetPlaybackInfo().PlaybackStatus.ToString()
    $entry = [pscustomobject]@{ app = $session.SourceAppUserModelId; title = Get-Title $session }

    if ($env:HEADS_UP_ACTION -eq 'pause' -and $status -eq 'Playing') {
        if (Await ($session.TryPauseAsync()) ([bool])) { $done += $entry }
    }
    elseif ($env:HEADS_UP_ACTION -eq 'resume' -and $status -eq 'Paused') {
        $match = $targets | Where-Object { $_.app -eq $entry.app -and $_.title -eq $entry.title }
        if ($match -and (Await ($session.TryPlayAsync()) ([bool]))) { $done += $entry }
    }
}

# -InputObject keeps a one-item result as an array in PowerShell 5.1.
ConvertTo-Json -InputObject @($done) -Compress
