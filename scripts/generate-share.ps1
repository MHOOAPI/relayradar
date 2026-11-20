param(
  [string]$BaseUrl = 'http://127.0.0.1:8080',
  [string]$OutDir = (Join-Path $PSScriptRoot '..\web\public\share')
)

Add-Type -AssemblyName System.Drawing
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

$W = 1200; $H = 630
$accent = [System.Drawing.Color]::FromArgb(46, 230, 166)
$white = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(236, 241, 248))
$gray = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(150, 162, 180))
$dim = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(110, 122, 140))
$bg1 = [System.Drawing.Color]::FromArgb(7, 10, 14)
$bg2 = [System.Drawing.Color]::FromArgb(12, 18, 26)

$fBrand = New-Object System.Drawing.Font('Segoe UI', 26, [System.Drawing.FontStyle]::Bold, [System.Drawing.GraphicsUnit]::Pixel)
$fBrandCn = New-Object System.Drawing.Font('Microsoft YaHei', 15, [System.Drawing.FontStyle]::Regular, [System.Drawing.GraphicsUnit]::Pixel)
$fBadge = New-Object System.Drawing.Font('Consolas', 20, [System.Drawing.FontStyle]::Bold, [System.Drawing.GraphicsUnit]::Pixel)
$fTitle = New-Object System.Drawing.Font('Microsoft YaHei', 46, [System.Drawing.FontStyle]::Bold, [System.Drawing.GraphicsUnit]::Pixel)
$fSub = New-Object System.Drawing.Font('Consolas', 22, [System.Drawing.FontStyle]::Regular, [System.Drawing.GraphicsUnit]::Pixel)
$fBig = New-Object System.Drawing.Font('Consolas', 108, [System.Drawing.FontStyle]::Bold, [System.Drawing.GraphicsUnit]::Pixel)
$fLabel = New-Object System.Drawing.Font('Microsoft YaHei', 14, [System.Drawing.FontStyle]::Regular, [System.Drawing.GraphicsUnit]::Pixel)
$fValue = New-Object System.Drawing.Font('Consolas', 26, [System.Drawing.FontStyle]::Bold, [System.Drawing.GraphicsUnit]::Pixel)
$fFoot = New-Object System.Drawing.Font('Microsoft YaHei', 13, [System.Drawing.FontStyle]::Regular, [System.Drawing.GraphicsUnit]::Pixel)

function New-Card {
  param($Path, $Title, $Subtitle, $Big, $Stats, $Badge, $Note)

  $bmp = New-Object System.Drawing.Bitmap($W, $H)
  $g = [System.Drawing.Graphics]::FromImage($bmp)
  $g.SmoothingMode = 'AntiAlias'
  $g.TextRenderingHint = 'ClearTypeGridFit'
  $rect = New-Object System.Drawing.Rectangle(0, 0, $W, $H)
  $grad = New-Object System.Drawing.Drawing2D.LinearGradientBrush($rect, $bg1, $bg2, 90)
  $g.FillRectangle($grad, $rect)

  $accentBrush = New-Object System.Drawing.SolidBrush($accent)
  $g.FillRectangle($accentBrush, 0, 0, 6, $H)

  # v2 material: soft glow + fine grid, matching the site.
  $glowPath = New-Object System.Drawing.Drawing2D.GraphicsPath
  $glowPath.AddEllipse(-300, -360, 980, 700)
  $glowBrush = New-Object System.Drawing.Drawing2D.PathGradientBrush($glowPath)
  $glowBrush.CenterColor = [System.Drawing.Color]::FromArgb(26, 46, 230, 166)
  $glowBrush.SurroundColors = @([System.Drawing.Color]::FromArgb(0, 46, 230, 166))
  $g.FillEllipse($glowBrush, -300, -360, 980, 700)
  $gridPen = New-Object System.Drawing.Pen([System.Drawing.Color]::FromArgb(11, 255, 255, 255), 1)
  for ($gx = 0; $gx -lt $W; $gx += 48) { $g.DrawLine($gridPen, $gx, 0, $gx, $H) }
  for ($gy = 0; $gy -lt $H; $gy += 48) { $g.DrawLine($gridPen, 0, $gy, $W, $gy) }

  # brand
  $g.DrawString('RelayRadar', $fBrand, $white, 64, 44)
  $g.DrawString('中转雷达', $fBrandCn, $gray, 66, 80)
  $g.DrawString('relay index', $fBrandCn, $dim, 66, 100)

  # badge
  if ($Badge) {
    $bs = $g.MeasureString($Badge, $fBadge)
    $bx = $W - 64 - $bs.Width - 28
    $penA = New-Object System.Drawing.Pen($accent, 1.2)
    $g.DrawRectangle($penA, $bx, 52, $bs.Width + 26, $bs.Height + 10)
    $g.DrawString($Badge, $fBadge, $accentBrush, $bx + 13, 58)
  }

  # title
  $ts = $g.MeasureString($Title, $fTitle)
  $titleSize = 46
  while ($ts.Width -gt 1040 -and $titleSize -gt 26) {
    $titleSize -= 2
    $fTitle = New-Object System.Drawing.Font('Microsoft YaHei', $titleSize, [System.Drawing.FontStyle]::Bold, [System.Drawing.GraphicsUnit]::Pixel)
    $ts = $g.MeasureString($Title, $fTitle)
  }
  $g.DrawString($Title, $fTitle, $white, 64, 180)
  $g.DrawString($Subtitle, $fSub, $gray, 66, 246)

  # big number
  $g.DrawString($Big, $fBig, $accentBrush, 58, 288)

  # stats
  $x = 66
  $colW = 268
  foreach ($st in $Stats) {
    $parts = $st -split '\|'
    $g.DrawString($parts[0], $fLabel, $dim, $x, 470)
    $g.DrawString($parts[1], $fValue, $white, $x, 492)
    $x += $colW
  }

  # footer
  $pen = New-Object System.Drawing.Pen([System.Drawing.Color]::FromArgb(30, 36, 46), 1)
  $g.DrawLine($pen, 64, 570, $W - 64, 570)
  $g.DrawString('relayradar.sh · independent relay monitoring', $fFoot, $dim, 64, 584)
  $g.DrawString($Note, $fFoot, $dim, $W - 64 - ($g.MeasureString($Note, $fFoot).Width), 584)

  $bmp.Save($Path, [System.Drawing.Imaging.ImageFormat]::Png)
  $g.Dispose(); $bmp.Dispose(); $grad.Dispose()
}

$stations = (Invoke-RestMethod "$BaseUrl/api/v1/stations").items
foreach ($s in $stations) {
  $stats = @(
    "可用性|$([math]::Round($s.uptimePct,1))%",
    "速度分|$([math]::Round($s.speed,1))",
    "稳定性|$([math]::Round($s.consistency,1))",
    "最低输入|¥$([math]::Round($s.fromPriceCny,2))/M"
  )
  New-Card -Path (Join-Path $OutDir "station-$($s.slug).png") -Title $s.name -Subtitle $s.domain `
    -Big ("{0:N1}" -f $s.rankScore) -Stats $stats -Badge "#$($s.rank) / $($stations.Count)" -Note 'demo dataset'
}

$models = (Invoke-RestMethod "$BaseUrl/api/v1/models").items
foreach ($m in $models) {
  $ctx = if ($m.contextWindow -ge 1000000) { "$([math]::Round($m.contextWindow/1000000,1))M" } else { "$([math]::Round($m.contextWindow/1000))K" }
  $inPrice = '$' + $m.inputUsd + '/M'
  $outPrice = '$' + $m.outputUsd + '/M'
  $stats = @(
    "官方输入|$inPrice",
    "官方输出|$outPrice",
    "全网最低|¥$([math]::Round($m.minInputCny,2))/M",
    "报价站点|$($m.offersCount)"
  )
  New-Card -Path (Join-Path $OutDir "model-$($m.id).png") -Title $m.name -Subtitle "$($m.vendorName) · $ctx context · $($m.license)" `
    -Big ("{0:N1}" -f $m.codingScore) -Stats $stats -Badge 'MODEL' -Note 'demo dataset'
}

"S cards=" + $stations.Count + " models=" + $models.Count
