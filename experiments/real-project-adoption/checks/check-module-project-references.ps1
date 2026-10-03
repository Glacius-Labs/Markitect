[CmdletBinding()]
param([Parameter(Position=0)][string]$ProjectRoot=(Get-Location).Path,[switch]$Control)
$ErrorActionPreference='Stop'
$mapPath=Join-Path $PSScriptRoot 'project-map.json'
$map=Get-Content -LiteralPath $mapPath -Raw | ConvertFrom-Json
$root=(Get-Item -LiteralPath $ProjectRoot).FullName
$manifestPath=Join-Path $root 'source-manifest.json'
if(!(Test-Path -LiteralPath $manifestPath -PathType Leaf)){throw "Source manifest is required: $manifestPath"}
$manifest=Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
$revision=[string]$manifest.upstreamCommit
if($revision -ne $map.upstreamCommit){throw "Upstream identity mismatch: expected $($map.upstreamCommit), got $revision"}
$manifestHash=(Get-FileHash -LiteralPath $manifestPath -Algorithm SHA256).Hash.ToLowerInvariant()
$snapshotRevision=$null
try{$gitHead=(& git -C $root rev-parse HEAD 2>$null|Out-String).Trim();if($LASTEXITCODE -eq 0){$snapshotRevision=$gitHead}}catch{}
$manifestFiles=@{}
foreach($entry in $manifest.files){$manifestFiles[[string]$entry.path]=$entry}
foreach($p in $map.projects){if(!$manifestFiles.ContainsKey([string]$p.path)){throw "Mapped project is absent from source manifest: $($p.path)"}}
$byId=@{}; $byPath=@{}
foreach($p in $map.projects){
  $p | Add-Member fullPath ([IO.Path]::GetFullPath((Join-Path $root $p.path)))
  if(!(Test-Path -LiteralPath $p.fullPath -PathType Leaf)){throw "Mapped project missing: $($p.path)"}
  if($byId.ContainsKey($p.id)){throw "Duplicate project id: $($p.id)"}
  $byId[$p.id]=$p; $byPath[$p.fullPath]=$p
}
$tmp=$null; $controlSource=$null; $controlTarget=$null
if($Control){
  $controlSource=$byId[[string]$map.control.sourceProject]; $controlTarget=$byId[[string]$map.control.targetProject]
  if(!$controlSource -or !$controlTarget){throw 'Control source and target must be explicitly mapped.'}
  $tmp=Join-Path ([IO.Path]::GetTempPath()) ('markitect-reference-control-'+[guid]::NewGuid().ToString('N')+'.targets')
  $escaped=[Security.SecurityElement]::Escape($controlTarget.fullPath)
  [IO.File]::WriteAllText($tmp,"<Project><ItemGroup><ProjectReference Include=`"$escaped`" /></ItemGroup></Project>",[Text.UTF8Encoding]::new($false))
}
$edges=[Collections.Generic.List[object]]::new(); $projects=[Collections.Generic.List[object]]::new()
Push-Location (Join-Path $root 'src')
$version=(& dotnet --version 2>&1|Out-String).Trim()
if($LASTEXITCODE -ne 0){if($tmp -and (Test-Path -LiteralPath $tmp)){Remove-Item -LiteralPath $tmp -Force};Pop-Location;throw "Cannot read project-pinned SDK version: $version"}
try{
 foreach($p in @($map.projects | Sort-Object id)){
  $args=@('msbuild',$p.fullPath,'-nologo','-getItem:ProjectReference',"-p:Configuration=$($map.configuration)")
  $inject=($Control -and $p.id -eq $controlSource.id)
  if($inject){$args += "-p:CustomAfterMicrosoftCommonTargets=$tmp"}
  $lines=& dotnet @args 2>&1; $code=$LASTEXITCODE; $raw=($lines|Out-String).Trim()
  if($code -ne 0){throw "MSBuild evaluation failed for $($p.path) (exit $code): $raw"}
  try{$eval=$raw|ConvertFrom-Json}catch{throw "MSBuild did not return JSON for $($p.path): $raw"}
  $refs=@(); if($null -ne $eval.Items -and $null -ne $eval.Items.ProjectReference){$refs=@($eval.Items.ProjectReference)}
  $projects.Add([ordered]@{id=$p.id;resource=$p.owner;layer=$p.layer;path=$p.path.Replace('\','/');projectReferenceCount=$refs.Count;controlInjectionApplied=[bool]$inject})
  foreach($r in $refs){
   $targetPath=[IO.Path]::GetFullPath([string]$r.FullPath)
   if(!(Test-Path -LiteralPath $targetPath -PathType Leaf)){throw "Evaluated reference is missing: $($p.path) -> $targetPath"}
   if(!$byPath.ContainsKey($targetPath)){throw "Evaluated reference target is unmapped: $($p.path) -> $targetPath"}
   $t=$byPath[$targetPath]; $status='allowed'; $why='same owner or inward to Core'
   if($p.ownerKind -eq 'core' -and $t.ownerKind -eq 'module'){$status='violation';$why='Core/BuildingBlocks must not reference a Module'}
   elseif($p.ownerKind -eq 'module' -and $t.ownerKind -eq 'module' -and $p.owner -ne $t.owner){
    if($map.allowedCrossModuleTargetLayers -contains $t.layer){$why='cross-module reference targets the allowed IntegrationEvents contract layer'}
    else{$status='violation';$why='cross-module reference must target the allowed IntegrationEvents contract layer'}
   }
   $origin=[string]$r.DefiningProjectFullPath
   if($origin){try{$origin=[IO.Path]::GetRelativePath($root,$origin).Replace('\','/')}catch{}}
   $edges.Add([ordered]@{source=$p.id;sourceResource=$p.owner;sourceLayer=$p.layer;sourcePath=$p.path.Replace('\','/');target=$t.id;targetResource=$t.owner;targetLayer=$t.layer;targetPath=$t.path.Replace('\','/');evaluatedIdentity=[string]$r.Identity;definingProject=$origin;status=$status;explanation=$why;controlInjected=[bool]($inject -and $t.id -eq $controlTarget.id)})
  }
 }
}finally{if($tmp -and (Test-Path -LiteralPath $tmp)){Remove-Item -LiteralPath $tmp -Force};Pop-Location}
$edges=@($edges|Sort-Object source,target,evaluatedIdentity); $findings=@($edges|Where-Object status -eq 'violation')
$controlPassed=$false
if($Control){$injectedFindings=@($findings|Where-Object controlInjected); $unexpectedFindings=@($findings|Where-Object { -not $_.controlInjected }); $controlPassed=($injectedFindings.Count -eq 1 -and $unexpectedFindings.Count -eq 0);if(!$controlPassed){$findings+=@([ordered]@{source=$controlSource.id;target=$controlTarget.id;status='control-failure';explanation='MSBuild did not expose injected prohibited ProjectReference'})}}


$inputSet=[Collections.Generic.SortedSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
$inputSet.Add($mapPath)|Out-Null; $inputSet.Add($PSCommandPath)|Out-Null; $inputSet.Add($manifestPath)|Out-Null
foreach($p in $map.projects){$inputSet.Add($p.fullPath)|Out-Null}
foreach($rel in @('src/Directory.Build.props','src/Directory.Build.targets','src/Directory.Packages.props','src/global.json')){$f=Join-Path $root $rel;if(Test-Path -LiteralPath $f -PathType Leaf){$inputSet.Add($f)|Out-Null}}
$inputs=@(foreach($f in $inputSet){[ordered]@{path=[IO.Path]::GetRelativePath($root,$f).Replace('\','/');sha256=(Get-FileHash -LiteralPath $f -Algorithm SHA256).Hash.ToLowerInvariant()}})
$report=[ordered]@{
 evidenceType='evaluated-dotnet-project-references';evidenceScope='Only explicitly mapped Module Application/Domain/IntegrationEvents projects and BuildingBlocks Application/Domain/Infrastructure.';upstreamCommit=$revision;snapshotRevision=$snapshotRevision;sourceManifestSha256=$manifestHash;configuration=[string]$map.configuration;dotnetSdk=$version
 evaluatedProjectCount=$projects.Count;projects=@($projects|Sort-Object id);referenceEdgeCount=$edges.Count;edges=$edges
 policy=[ordered]@{crossModuleTargetLayers=@($map.allowedCrossModuleTargetLayers|Sort-Object);coreMayReferenceModules=$false};findings=$findings
 control=[ordered]@{enabled=[bool]$Control;injectedEdgeDetectedAndRejected=[bool]$controlPassed;scratchTargetLocation=if($Control){'OS temporary directory; removed after evaluation'}else{$null}}
 inputs=$inputs
 proves=@('MSBuild evaluated ProjectReference items under the recorded revision, SDK, configuration, and hashed project/import inputs.','The listed edges were checked against the explicit owner and layer policy.')
 doesNotProve=@('Runtime DI, reflection, plugin loading, network calls, or other dependency mechanisms.','That IntegrationEvents contains only stable public contracts or that a referenced contract is semantically appropriate.','Behavior, business meaning, unlisted projects, or architecture outside the explicit map.')
}
$report|ConvertTo-Json -Depth 20
if($Control){if(!$controlPassed){exit 2};exit 0};if($findings.Count -gt 0){exit 1};exit 0