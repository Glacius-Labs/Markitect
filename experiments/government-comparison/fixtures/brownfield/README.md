# Brownfield Orders API fixture

This is the existing service snapshot for the study's Brownfield condition. The experiment coordinator supplies the public behavior brief and shared checks separately from this actor repository.

From this directory, restore the exact dependency graph and build:

```powershell
dotnet restore src/Orders.Api/Orders.Api.csproj --locked-mode
dotnet build src/Orders.Api/Orders.Api.csproj --no-restore
```

Run it with an isolated database path and listener:

```powershell
$databaseDirectory = Join-Path (Get-Location) '.study'
New-Item -ItemType Directory -Force $databaseDirectory | Out-Null
$env:ORDERS_DB = Join-Path $databaseDirectory 'orders.db'
$env:ASPNETCORE_URLS = 'http://127.0.0.1:5187'
dotnet run --no-launch-profile --project src/Orders.Api/Orders.Api.csproj
```

Run the separately supplied public smoke checks using their absolute path:

```powershell
python C:\study\public\checks.py --base-url http://127.0.0.1:5187 --condition brownfield --through-task 1
```

Replace `C:\study\public\checks.py` with the absolute path where the coordinator mounted the shared checks. Task 1 checks the pre-existing order behavior and compatibility route. The fixture intentionally has no stock reservation or lifecycle behavior at this starting point.
