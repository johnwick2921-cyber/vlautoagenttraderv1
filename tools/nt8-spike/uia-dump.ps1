# W-ONE-BUTTON M1 step 1 - READ-ONLY UI Automation tree dump of NinjaTrader 8 top-level windows.
# Property reads ONLY. No Invoke, no SetFocus, no Expand/Collapse, no Select, no Scroll, no pattern calls.
# Grid/list/table/tree/document/edit subtrees are NOT descended (account names and values never enter the file).
# Any Name with 4+ consecutive digits is masked. The repo is public.
param(
  [int]$TargetPid = 0,
  [int]$MaxDepth = 14,
  [int]$MaxNodes = 4000,
  [int]$TimeoutSec = 120
)
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
$AE = [System.Windows.Automation.AutomationElement]
$CT = [System.Windows.Automation.ControlType]

if ($TargetPid -le 0) {
  $p = Get-Process NinjaTrader -ErrorAction Stop | Select-Object -First 1
  $TargetPid = $p.Id
}
$proc = Get-Process -Id $TargetPid -ErrorAction Stop
$skipTypes = @($CT::DataGrid, $CT::DataItem, $CT::List, $CT::Table, $CT::Tree, $CT::Document, $CT::Edit)
# Names are kept ONLY for interactive UI labels; every other element records its name LENGTH only
# (account names, connection names and values never enter the file).
$nameTypes = @($CT::Window, $CT::Menu, $CT::MenuItem, $CT::MenuBar, $CT::Button, $CT::SplitButton, $CT::TabItem, $CT::ToolBar, $CT::Header, $CT::HeaderItem, $CT::CheckBox, $CT::RadioButton, $CT::Hyperlink, $CT::TitleBar)
$sw = [System.Diagnostics.Stopwatch]::StartNew()
$script:nodes = 0
$out = New-Object System.Collections.Generic.List[string]

function Mask([string]$s) {
  if ($null -eq $s) { return '' }
  $s = $s -replace '\d{4,}', '<masked>'
  $s = $s -replace '[\r\n\t]', ' '
  if ($s.Length -gt 80) { $s = $s.Substring(0, 80) + '...' }
  return $s
}

function Patterns($el) {
  try { ($el.GetSupportedPatterns() | ForEach-Object { $_.ProgrammaticName -replace 'PatternIdentifiers.Pattern','' }) -join ',' } catch { '?' }
}

$walker = [System.Windows.Automation.TreeWalker]::ControlViewWalker

function Walk($el, [int]$depth) {
  if ($script:nodes -ge $MaxNodes -or $sw.Elapsed.TotalSeconds -ge $TimeoutSec) { return }
  $script:nodes++
  try {
    $c = $el.Current
    $ct = $c.ControlType.ProgrammaticName -replace 'ControlType.',''
    $nm = if ($nameTypes -contains $c.ControlType) { Mask $c.Name } elseif ([string]::IsNullOrEmpty($c.Name)) { '' } else { '<len=' + $c.Name.Length + '>' }
    $line = ('  ' * $depth) + "[$ct] aid='" + (Mask $c.AutomationId) + "' name='" + $nm + "' class='" + (Mask $c.ClassName) + "' fw='" + $c.FrameworkId + "' enabled=" + $c.IsEnabled + " patterns=" + (Patterns $el)
    $gridLike = ($c.ControlType -eq $CT::Custom -and $depth -gt 0 -and ($c.ClassName -match 'Grid|Account|Position|Order|Execution|Log'))
    if (($skipTypes -contains $c.ControlType) -or $gridLike) {
      $n = 0; $ch = $walker.GetFirstChild($el); while ($ch -ne $null -and $n -lt 10000) { $n++; $ch = $walker.GetNextSibling($ch) }
      $out.Add($line + " <subtree skipped: $ct, children=$n>")
      return
    }
    $out.Add($line)
  } catch { $out.Add(('  ' * $depth) + "<read error: $($_.Exception.GetType().Name)>"); return }
  if ($depth -ge $MaxDepth) { $out.Add(('  ' * ($depth+1)) + '<depth cap>'); return }
  $child = $walker.GetFirstChild($el)
  while ($child -ne $null) {
    Walk $child ($depth + 1)
    if ($script:nodes -ge $MaxNodes -or $sw.Elapsed.TotalSeconds -ge $TimeoutSec) { break }
    $child = $walker.GetNextSibling($child)
  }
}

$cond = New-Object System.Windows.Automation.PropertyCondition($AE::ProcessIdProperty, $TargetPid)
$tops = $AE::RootElement.FindAll([System.Windows.Automation.TreeScope]::Children, $cond)
$out.Add("# UIA dump - NinjaTrader $($proc.MainModule.FileVersionInfo.FileVersion) pid=$TargetPid start=$($proc.StartTime.ToString('o'))")
$out.Add("# read-only; skipped subtree types: DataGrid DataItem List Table Tree Document Edit + Custom classes matching Grid|Account|Position|Order|Execution|Log; names kept only for interactive UI labels (others: length only); 4+ digits masked")
$out.Add("# top-level windows for pid: $($tops.Count)")
foreach ($t in $tops) { Walk $t 0 }
$out.Add("# nodes=$($script:nodes) elapsed_s=$([math]::Round($sw.Elapsed.TotalSeconds,1)) capped=$(($script:nodes -ge $MaxNodes) -or ($sw.Elapsed.TotalSeconds -ge $TimeoutSec))")
$out -join "`n"
