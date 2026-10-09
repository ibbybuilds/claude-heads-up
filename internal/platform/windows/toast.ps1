# Shows a silent Windows toast (the alert sound is played separately, so the
# user can turn either one off).
#
# Input: $env:HEADS_UP_TITLE, $env:HEADS_UP_BODY, $env:HEADS_UP_STICKY ('1' or '0')
#
# Sticky uses the 'reminder' scenario, which keeps the toast on screen until
# dismissed. Windows ignores the scenario unless the toast has a button.

$ErrorActionPreference = 'Stop'
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom, ContentType = WindowsRuntime] | Out-Null

$sticky = $env:HEADS_UP_STICKY -eq '1'
$scenario = if ($sticky) { " scenario='reminder'" } else { '' }
$actions = if ($sticky) { "<actions><action activationType='system' arguments='dismiss' content=''/></actions>" } else { '' }

$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml("<toast$scenario><visual><binding template='ToastGeneric'><text/><text/></binding></visual>$actions<audio silent='true'/></toast>")
$texts = $xml.GetElementsByTagName('text')
$texts.Item(0).AppendChild($xml.CreateTextNode($env:HEADS_UP_TITLE)) | Out-Null
$texts.Item(1).AppendChild($xml.CreateTextNode($env:HEADS_UP_BODY)) | Out-Null

# An unpackaged app needs a registered AppUserModelID to show toasts. Windows
# PowerShell's own ID is always registered, so the toast is attributed to it.
$appId = '{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe'
$toast = New-Object Windows.UI.Notifications.ToastNotification $xml
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($appId).Show($toast)
