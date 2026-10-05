param(
 [Parameter(Mandatory=$true)][string]$Repo,
 [Parameter(Mandatory=$true)][ValidatePattern('^(0[1-9]|1[0-2]|P0[12])$')][string]$Task,
 [Parameter(Mandatory=$true)][string]$Output,
 [string]$Base = $env:GAUNTLET_TASK_BASE,
 [ValidatePattern('^0[0-6]$')][string]$PriorTasksThrough = '06'
)
$ErrorActionPreference='Stop'
$script:repoPath=(Resolve-Path -LiteralPath $Repo).Path
$script:oracleRoot=$PSScriptRoot
$script:checks=[System.Collections.Generic.List[object]]::new()
$script:failures=[System.Collections.Generic.List[string]]::new()
$script:commands=[System.Collections.Generic.List[string]]::new()
$script:arm=if(Test-Path (Join-Path $script:repoPath 'markitect.yaml')){'b'}else{'a'}
$script:isParallel=$Task.StartsWith('P',[StringComparison]::Ordinal)
$script:originTask=switch($Task){'P01'{'07'};'P02'{'08'};default{$Task}}
$script:sourceThrough=if($script:isParallel){[int]$PriorTasksThrough}else{[int]$script:originTask}

function Add-Check([string]$Name,[bool]$Passed,[string]$Evidence){$script:checks.Add([pscustomobject]@{name=$Name;passed=$Passed;evidence=$Evidence});if(-not $Passed){$script:failures.Add($Name)}}
function Read-SeedFile([string]$Relative){$p=Join-Path $script:repoPath $Relative;if(Test-Path -LiteralPath $p -PathType Leaf){return Get-Content -LiteralPath $p -Raw};return ''}
function Has-Text([string]$Path,[string]$Pattern){return (Read-SeedFile $Path) -match $Pattern}
function Has-CompleteOwnershipTable([string]$Text){
 $lines=$Text -split "`r?`n"
 $seen=@{}
 foreach($line in $lines){
  $line=$line.Trim()
  if($line -notmatch '^\|.*\|$'){continue}
  $cells=@($line.Trim('|').Split('|')|ForEach-Object{$_.Trim()})
  $moduleCells=@($cells|Where-Object{$_ -match '^(?i)(Orders|Inventory|Billing)$'})
  if($moduleCells.Count -eq 0){continue}
  if($moduleCells.Count -ne 1){return $false}
  $module=switch -Regex ($moduleCells[0]){'(?i)^Orders$'{'Orders';break}'(?i)^Inventory$'{'Inventory';break}'(?i)^Billing$'{'Billing';break}}
  if($seen.ContainsKey($module)){return $false}
  $seen[$module]=$true
  $ownershipCells=@()
  foreach($cell in $cells){
   # Recognize the bounded ownership claims and the observed Orders coordination clause; this is not general prose interpretation.
   $value=($cell -replace '[`*_]','').ToLowerInvariant() -replace '\s+',' '
   $value=$value.Trim() -replace '\s+and\s+(?=coordinates?\b)','; '
   $clauses=@($value -split '[;.!?]+'|ForEach-Object{$_.Trim()}|Where-Object{$_})
   $ownershipPattern=switch($module){
    'Orders' {'^(?:owns?\s+(?:the\s+)?order lifecycle(?:\s+and order decisions|\s+decisions)?|order lifecycle(?:\s+decisions)?(?:\s+owner)?)$'}
    'Inventory' {'^(?:owns?\s+(?:the\s+)?stock and reservations?|stock and reservations?(?:\s+owner)?)$'}
    'Billing' {'^(?:owns?\s+(?:the\s+)?invoice records and idempotency|invoice records and idempotency(?:\s+owner)?)$'}
    default {return $false}
   }
   $ownershipClaims=@($clauses|Where-Object{$_ -match $ownershipPattern})
   $validOwnership=$ownershipClaims.Count -eq 1
   if($validOwnership){
    $remaining=@($clauses|Where-Object{$_ -notmatch $ownershipPattern})
    $coordinationPattern='^coordinates?\s+(?:the\s+)?reservation and invoice requests through contracts$'
    if($module -ne 'Orders'){$validOwnership=$remaining.Count -eq 0}
    else{foreach($clause in $remaining){if($clause -notmatch $coordinationPattern){$validOwnership=$false;break}}}
   }
   if($validOwnership){$ownershipCells+=$cell}
  }
  if($ownershipCells.Count -ne 1){return $false}
  $packageCells=@($cells|Where-Object{($_ -replace '[`*_]','').Trim() -match '(?i)^internal/modules/'})
  if($packageCells.Count -ne 1){return $false}
  $package=($packageCells[0] -replace '[`*_]','').Trim()
  if($package -cne ('internal/modules/'+$module.ToLowerInvariant())){return $false}

  $dependencyCells=@($cells|Where-Object{$_ -match '(?i)\bCore\b' -and $_ -match '(?i)\bContracts\b'})
  if($dependencyCells.Count -ne 1){return $false}
  $dependencyText=($dependencyCells[0] -replace '[`*_]','').Trim()
  $dependencyMatches=[regex]::Matches($dependencyText,'(?i)\b(Core|Contracts)\b(?:\s*\(\s*([^()]*)\s*\))?')
  if($dependencyMatches.Count -ne 2){return $false}
  $found=@{}
  foreach($match in $dependencyMatches){
   $name=$match.Groups[1].Value.ToLowerInvariant()
   if($found.ContainsKey($name)){return $false}
   $path=$match.Groups[2]
   if($path.Success){
    $expectedPath=if($name -eq 'core'){'internal/core'}else{'internal/contracts'}
    if($path.Value.Trim() -cne $expectedPath){return $false}
   }
   $found[$name]=$true
  }
  if(-not $found.ContainsKey('core') -or -not $found.ContainsKey('contracts')){return $false}
  $remainder=$dependencyText
  foreach($match in $dependencyMatches){$position=$remainder.IndexOf($match.Value,[StringComparison]::Ordinal);if($position -lt 0){return $false};$remainder=$remainder.Remove($position,$match.Length)}
  $remainder=$remainder -replace '(?i)\band\b','' -replace '[,;&\s]',''
  if($remainder.Length -ne 0){return $false}
 }
 return $seen.Count -eq 3
}
function Has-MetadataName([string]$Text,[string]$Name){$metadata=[regex]::Match($Text,'(?ims)^metadata:\s*(?<body>.*?)(?=^spec:|\z)');return $metadata.Success -and $metadata.Groups['body'].Value -match ('(?im)^\s*name:\s*'+[regex]::Escape($Name)+'\s*$')}
function Has-Relation([string]$Text,[string]$Field,[string]$Kind,[string]$Name,[string]$Namespace){$lines=$Text -split "`r?`n";$fieldPattern='^(?<indent>\s*)'+[regex]::Escape($Field)+':\s*(?<value>.*)$';$kindPattern='(?im)(?:\bkind:\s*'+[regex]::Escape($Kind)+'\s*(?=,|\}|$)|^\s*kind:\s*'+[regex]::Escape($Kind)+'\s*$)';$namePattern='(?im)(?:\bname:\s*'+[regex]::Escape($Name)+'\s*(?=,|\}|$)|^\s*name:\s*'+[regex]::Escape($Name)+'\s*$)';$namespacePattern='(?im)(?:\bnamespace:\s*'+[regex]::Escape($Namespace)+'\s*(?=,|\}|$)|^\s*namespace:\s*'+[regex]::Escape($Namespace)+'\s*$)';for($i=0;$i -lt $lines.Count;$i++){$fieldMatch=[regex]::Match($lines[$i],$fieldPattern);if(-not $fieldMatch.Success){continue};$value=$fieldMatch.Groups['value'].Value;if($value -match '^\{.*\}$'){$block=$value}else{$indent=$fieldMatch.Groups['indent'].Value.Length;$parts=@();for($j=$i+1;$j -lt $lines.Count;$j++){if($lines[$j].Trim() -eq ''){continue};$nextIndent=([regex]::Match($lines[$j],'^\s*')).Length;if($nextIndent -le $indent){break};$parts+=$lines[$j]};$block=$parts -join "`n"};if($block -match $kindPattern -and $block -match $namePattern -and $block -match $namespacePattern){return $true}};return $false}
function Run-Go([string[]]$GoArgs){$script:commands.Add(('go '+($GoArgs -join ' ')));Push-Location $script:repoPath;try{& go @GoArgs *> $null;return $LASTEXITCODE -eq 0}finally{Pop-Location}}
function Run-Vector([int]$VectorTask,[string]$Dir,[string[]]$GoArgs){$from=Join-Path $PSScriptRoot ('vectors/task'+$VectorTask.ToString('00')+'_test.go.txt');$to=Join-Path (Join-Path $script:repoPath $Dir) 'gauntlet_hidden_test.go';if(Test-Path -LiteralPath $to){throw "Refusing to overwrite evaluator path $to"};$source=Get-Content -LiteralPath $from -Raw;if($VectorTask -eq 2 -and [int]$script:originTask -ge 4){$source=$source.Replace('IssueRequest','CreateInvoiceRequest')};if($VectorTask -eq 1 -and [int]$script:originTask -ge 11){$source=$source.Replace('Quantity','Units')};[IO.File]::WriteAllText($to,$source,[Text.UTF8Encoding]::new($false));try{return Run-Go $GoArgs}finally{Remove-Item -LiteralPath $to -Force}}
function Get-CardScopes([string]$TaskId,[string]$Arm,[bool]$Parallel){$file=if($Parallel){'..\parallel-task-set.yaml'}else{'..\task-set.yaml'};$yaml=Get-Content -LiteralPath (Join-Path $script:oracleRoot $file) -Raw;$match=[regex]::Match($yaml,'(?ms)^  - id: "?'+[regex]::Escape($TaskId)+'"?\r?\n(?<body>.*?)(?=^  - id:|\z)');if(-not $match.Success){throw "Task $TaskId not found in $file"};$body=$match.Groups['body'].Value;$armLine=[regex]::Match($body,'(?m)^      '+[regex]::Escape($Arm)+': \[(?<items>[^\]]*)\]');$line=if($armLine.Success){$armLine.Groups['items'].Value}else{([regex]::Match($body,'(?m)^    allowed_paths: \[(?<items>[^\]]*)\]')).Groups['items'].Value};if([string]::IsNullOrWhiteSpace($line)){return @()};return @($line -split ',\s*'|ForEach-Object{$_.Trim().Trim('"').Trim("'")}|Where-Object{$_})}
function Test-ParallelIntegration([string]$TaskId,[string]$PriorTasksThrough,[bool]$PriorTasksThroughWasExplicit){return $TaskId -ceq '08' -and $PriorTasksThroughWasExplicit -and $PriorTasksThrough -ceq '06'}
function Get-EffectiveCardScopes([string]$TaskId,[string]$Arm,[bool]$Parallel,[bool]$ParallelIntegration){if($ParallelIntegration){if($TaskId -cne '08'){throw 'Parallel integration scope applies only to combined task 08'};$scopes=@(Get-CardScopes 'P01' $Arm $true);$scopes+=@(Get-CardScopes 'P02' $Arm $true);return $scopes};return @(Get-CardScopes $TaskId $Arm $Parallel)}

$parallelIntegration=Test-ParallelIntegration $Task $PriorTasksThrough $PSBoundParameters.ContainsKey('PriorTasksThrough')
$scopes=Get-EffectiveCardScopes $Task $script:arm $script:isParallel $parallelIntegration
$changed=@()
if($Base){Push-Location $script:repoPath;try{$script:commands.Add("git diff --name-only $Base plus ordinary and ignored untracked paths");$changed+=@(git diff --name-only $Base);$changed+=@(git ls-files --others --exclude-standard);$changed+=@(git ls-files --others --ignored --exclude-standard|Where-Object{$_ -notmatch '^\.cache/'})}finally{Pop-Location};$bad=@($changed|Sort-Object -Unique|Where-Object{$p=$_ -replace '\\','/';$allowed=$false;foreach($scope in $scopes){$s=$scope -replace '\\','/';if($s.EndsWith('/')){if($p.StartsWith($s,[StringComparison]::OrdinalIgnoreCase)){$allowed=$true;break}}elseif($p.Equals($s,[StringComparison]::OrdinalIgnoreCase)){$allowed=$true;break}};-not $allowed});Add-Check 'allowed-paths' ($bad.Count -eq 0) (($changed|Sort-Object -Unique)-join ',')}
else{Add-Check 'allowed-paths' $false 'Supply the task-start commit with -Base or GAUNTLET_TASK_BASE.'}

$arch=Run-Go @('test','./internal/architecture');Add-Check 'architecture-boundary' $arch 'go test ./internal/architecture'
if($script:originTask -eq '10'){Add-Check 'no-auto-implementation' ($changed.Count -eq 0) 'No changed or untracked project paths';Add-Check 'owner-decision-pending' $true 'Result remains pending a human owner decision; no waiver can satisfy it.'}

if($script:sourceThrough -ge 3){$d=Read-SeedFile 'docs/architecture.md';Add-Check 'cumulative-ownership-table' (Has-CompleteOwnershipTable $d) 'Every module has responsibility, allowed dependencies, and package path'}
if($script:sourceThrough -ge 4){$all=((Read-SeedFile 'internal/contracts/billing/invoice.go')+(Read-SeedFile 'internal/modules/billing/service.go')+(Read-SeedFile 'internal/modules/billing/service_test.go')+(Read-SeedFile 'internal/modules/orders/service.go')+(Read-SeedFile 'internal/modules/orders/service_test.go'));Add-Check 'cumulative-contract-rename' ($all -match '\bCreateInvoiceRequest\b' -and $all -notmatch '\bIssueRequest\b') 'Renamed billing contract remains canonical'}
if($script:sourceThrough -ge 6){if($script:arm -eq 'b'){$u=Read-SeedFile 'resources/usecase-get-availability.yaml';$intent=(Has-MetadataName $u 'get-availability') -and (Has-Relation $u 'module' 'Module' 'inventory' 'architecture') -and ($u -match '(?is)GetAvailability')}else{$d=Read-SeedFile 'docs/architecture.md';$intent=(($d -match '(?is)Inventory.{0,200}GetAvailability') -or ($d -match '(?is)GetAvailability.{0,200}Inventory'))};$guidance=if($script:arm -eq 'b'){Has-Text '.markitect/areas/constitution/implement-change.skill.yaml' 'get-availability'}else{Has-Text 'AGENTS.md' '(?is)Inventory.{0,200}GetAvailability|GetAvailability.{0,200}Inventory'};Add-Check 'cumulative-availability-intent' $intent 'Availability is assigned to Inventory in the arm owner record';Add-Check 'cumulative-availability-guidance' $guidance 'Availability ownership remains in selected or human-owned guidance'}
if($script:sourceThrough -ge 9){if($script:arm -eq 'b'){$u=Read-SeedFile 'resources/usecase-refund-invoice.yaml';$i=Read-SeedFile 'resources/interface-invoice-refund.yaml';$skill=Read-SeedFile '.markitect/areas/constitution/implement-change.skill.yaml';$intent=(Has-MetadataName $u 'refund-invoice') -and (Has-Relation $u 'module' 'Module' 'billing' 'architecture') -and ($u -match '(?is)refund') -and (Has-MetadataName $i 'invoice-refund') -and (Has-Relation $i 'provider' 'Module' 'billing' 'architecture');$guidance=(($skill -match '(?im)^[^.!?\r\n]*Billing[^.!?\r\n]{0,100}(owns?|owner of)[^.!?\r\n]{0,80}refund') -or ($skill -match '(?im)^[^.!?\r\n]*refund[^.!?\r\n]{0,80}(owned by|belongs to) Billing'))}else{$d=Read-SeedFile 'docs/architecture.md';$g=Read-SeedFile 'AGENTS.md';$intent=(($d -match '(?is)Billing.{0,200}owns.{0,200}refund') -or ($d -match '(?is)Billing.{0,200}refund.{0,200}owns'));$guidance=(($g -match '(?im)^[^.!?\r\n]*Billing[^.!?\r\n]{0,50}(owns?|owner of)[^.!?\r\n]{0,40}refund[^.!?\r\n]{0,100}(contract|interface)') -or ($g -match '(?im)^[^.!?\r\n]*refund[^.!?\r\n]{0,50}(contract|interface)[^.!?\r\n]{0,100}(owned by|belongs to) Billing'))};Add-Check 'cumulative-refund-intent' $intent 'Refund behavior is assigned to Billing in the human architecture record or typed owner records';Add-Check 'cumulative-refund-guidance' $guidance 'Selected engineering guidance directs refund contract work to Billing'}
if($script:sourceThrough -ge 11){$c=Read-SeedFile 'internal/contracts/inventory/reservation.go';$m=(Read-SeedFile 'internal/modules/inventory/service.go')+(Read-SeedFile 'internal/modules/orders/service.go');Add-Check 'cumulative-units-migration' ($c -match '\bUnits\b' -and $c -notmatch '\bQuantity\b' -and $m -match '\bUnits\b') 'Units contract and consumers remain canonical'}
if($script:sourceThrough -ge 12){$ci=Read-SeedFile '.github/workflows/ci.yml';$d=Read-SeedFile 'docs/architecture.md';$a=$ci.IndexOf('go test ./internal/architecture');$t=$ci.IndexOf('go test ./...');$namedFormat=$ci -match '(?im)^\s*-\s*name:\s*gofmt-check\s*\r?\n\s*run:\s*test -z "\$\(gofmt -l \.\)"\s*$';$contribution=[regex]::Match($d,'(?ims)^\s*#{1,6}\s*Contribution checks\s*(?<body>.*?)(?=^\s*#{1,6}\s|\z)');$body=$contribution.Groups['body'].Value;$commands=@('test -z "$(gofmt -l .)"','go test ./internal/architecture','go test ./...','go vet ./...');$documented=$contribution.Success;foreach($command in $commands){if(-not $body.Contains($command)){$documented=$false}};Add-Check 'named-failing-gofmt-check' $namedFormat 'Named step runs test -z "$(gofmt -l .)"';Add-Check 'vet-in-ci' $ci.Contains('go vet ./...') 'CI includes go vet ./...';Add-Check 'architecture-first' ($a -ge 0 -and $t -gt $a) 'Architecture test precedes full tests';Add-Check 'exact-contribution-commands' $documented 'All four exact local commands appear in Contribution checks'}

$vectorPlan=@(@{task=1;dir='internal/modules/inventory';test='TestGauntletIdempotencyVector';name='idempotency-vector'},@{task=2;dir='internal/modules/billing';test='TestGauntletInvoiceConflictVector';name='invoice-conflict-vector'},@{task=5;dir='internal/modules/orders';test='TestGauntletOverflowVector';name='overflow-vector'},@{task=6;dir='internal/modules/inventory';test='TestGauntletAvailabilityVector';name='availability-vector'},@{task=7;dir='internal/modules/orders';test='TestGauntletCustomerReferenceVector';name='customer-reference-vector'},@{task=8;dir='internal/modules/billing';test='TestGauntletMemoVector';name='invoice-memo-vector'},@{task=9;dir='internal/modules/billing';test='TestGauntletRefundVector';name='refund-vector'})
foreach($vector in $vectorPlan|Where-Object{if($script:isParallel){$_.task -le $script:sourceThrough -or $_.task -eq [int]$script:originTask}else{$_.task -le [int]$script:originTask}}){$ok=Run-Vector $vector.task $vector.dir @('test',("./"+$vector.dir.Replace('\','/')),'-run',('^'+$vector.test+'$'),'-count=1');Add-Check $vector.name $ok 'Cumulative fixed behavior vector'}
$valid=Run-Go @('test','./...');Add-Check 'task-validator' $valid 'go test ./...';if($script:originTask -eq '12'){$vet=Run-Go @('vet','./...');Add-Check 'go-vet' $vet 'go vet ./...'}
if($script:arm -eq 'b' -and $script:originTask -in @('06','09','11')){$exe=Get-Command markitect -ErrorAction SilentlyContinue;$helper=Get-Command markitect-check-artifacts -ErrorAction SilentlyContinue;if($exe -and $helper){Push-Location $script:repoPath;try{$candidate=(& git rev-parse HEAD).Trim();& markitect check --repo . --revision $candidate *> $null;$c=$LASTEXITCODE -eq 0;& markitect render --repo . --revision $candidate *> $null;$r=$LASTEXITCODE -eq 0;& markitect verify --repo . --revision $candidate *> $null;$v=$LASTEXITCODE -eq 0;& markitect-check-artifacts --repo . --config .markitect/coverage.yaml *> $null;$a=$LASTEXITCODE -eq 0}finally{Pop-Location};Add-Check 'markitect-check' $c 'Frozen CLI check';Add-Check 'generated-projections' $r 'Frozen CLI render check';Add-Check 'fixed-revision-verification' $v 'Frozen CLI verify';Add-Check 'artifact-coverage' $a 'Frozen artifact-coverage helper'}else{Add-Check 'markitect-check' $false 'Frozen candidate CLI and helper are unavailable on PATH'}}

$status=if($script:failures.Count -gt 0){'failed'}elseif($script:originTask -eq '10'){'owner-decision-required'}else{'passed'}
$checkLines=($script:checks|ForEach-Object{"  - name: $($_.name)`n    passed: $($_.passed.ToString().ToLowerInvariant())`n    evidence: `"$($_.evidence -replace '"','''')`""})-join "`n"
$failureLines=if($script:failures.Count){($script:failures|ForEach-Object{"  - $_"})-join "`n"}else{'  []'}
$baseText=if($Base){$Base}else{'unavailable'};$changedText=($changed|Sort-Object -Unique)-join ', '
$yaml="task: `"$Task`"`norigin_task: `"$script:originTask`"`narm: $script:arm`nstatus: $status`nbase: `"$baseText`"`nchanged_paths: `"$changedText`"`nchecks:`n$checkLines`nfailures:`n$failureLines`nowner_decision: $(if($script:originTask -eq '10'){'required; no automatic waiver'}else{'not required'})"
Set-Content -LiteralPath $Output -Value $yaml
$yaml
