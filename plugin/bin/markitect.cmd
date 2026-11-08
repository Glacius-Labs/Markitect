@echo off
rem markitect, as the Claude Code plugin supplies it to PowerShell and cmd.
rem
rem Claude Code puts this folder on the PATH of its PowerShell tool, which cannot
rem run the launcher beside this file, a POSIX shell script. This file runs the
rem launcher with the bash of Git for Windows, which Claude Code needs on Windows
rem too: the one CLAUDE_CODE_GIT_BASH_PATH names if it exists, or else the one that
rem belongs to the git.exe on the PATH. The bash.exe of Windows itself is WSL's and
rem is never used.
setlocal
set "MARKITECT_BASH=%CLAUDE_CODE_GIT_BASH_PATH%"
if defined MARKITECT_BASH if not exist "%MARKITECT_BASH%" set "MARKITECT_BASH="
if not defined MARKITECT_BASH for %%G in (git.exe) do if not "%%~$PATH:G"=="" for %%D in ("%%~dp$PATH:G.") do (
	if not defined MARKITECT_BASH if exist "%%~fD\..\bin\bash.exe" set "MARKITECT_BASH=%%~fD\..\bin\bash.exe"
	if not defined MARKITECT_BASH if exist "%%~fD\..\..\bin\bash.exe" set "MARKITECT_BASH=%%~fD\..\..\bin\bash.exe"
)
if not defined MARKITECT_BASH (
	>&2 echo markitect ^(plugin^): needs the bash of Git for Windows; set CLAUDE_CODE_GIT_BASH_PATH to its bash.exe
	exit /b 2
)
rem The launcher finds its own folder with dirname, which needs forward slashes.
set "MARKITECT_LAUNCHER=%~dp0markitect"
"%MARKITECT_BASH%" "%MARKITECT_LAUNCHER:\=/%" %*
