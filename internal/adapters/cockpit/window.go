package cockpit

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Launch is one way to open the cockpit's address: a program and its
// arguments.
type Launch struct {
	Name string
	Args []string
}

// WindowLaunches returns, in order of preference, the ways to open url in an
// app window of an installed Chromium browser (no tabs, no address bar) and,
// last, in the default browser. It only chooses; it starts nothing. getenv,
// lookPath and exists answer for the machine the cockpit runs on, which makes
// the choice testable for every platform. WSL_DISTRO_NAME marks a Linux that
// runs under Windows: the window then opens on the Windows side.
func WindowLaunches(goos string, getenv func(string) string, lookPath func(string) (string, error), exists func(string) bool, url string) []Launch {
	app := "--app=" + url
	var out []Launch
	found := func(name string) (string, bool) {
		path, err := lookPath(name)
		return path, err == nil && path != ""
	}
	switch goos {
	case "windows":
		for _, browser := range windowsBrowsers {
			for _, root := range []string{getenv("ProgramFiles"), getenv("ProgramFiles(x86)"), getenv("LocalAppData")} {
				if root == "" {
					continue
				}
				if path := root + `\` + browser.dir + `\Application\` + browser.exe; exists(path) {
					out = append(out, Launch{path, []string{app}})
				}
			}
			if path, ok := found(browser.exe); ok {
				out = append(out, Launch{path, []string{app}})
			}
		}
		out = append(out, Launch{"rundll32", []string{"url.dll,FileProtocolHandler", url}})
	case "darwin":
		home := getenv("HOME")
		for _, name := range []string{"Google Chrome", "Microsoft Edge"} {
			installed := exists("/Applications/"+name+".app") || home != "" && exists(home+"/Applications/"+name+".app")
			if installed {
				out = append(out, Launch{"open", []string{"-na", name, "--args", app}})
			}
		}
		out = append(out, Launch{"open", []string{url}})
	default:
		if getenv("WSL_DISTRO_NAME") != "" {
			// The Windows browsers are probed through the mount of the C drive and
			// run directly: starting through cmd.exe would succeed without a
			// browser and never reach the fallbacks.
			for _, path := range wslBrowsers {
				if exists(path) {
					out = append(out, Launch{path, []string{app}})
				}
			}
			if cmd, ok := found("cmd.exe"); ok {
				out = append(out, Launch{cmd, []string{"/c", "start", "", url}})
			}
		}
		for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "microsoft-edge", "microsoft-edge-stable"} {
			if path, ok := found(name); ok {
				out = append(out, Launch{path, []string{app}})
			}
		}
		if path, ok := found("xdg-open"); ok {
			out = append(out, Launch{path, []string{url}})
		}
	}
	return out
}

// wslBrowsers are the standard places of Edge and Chrome as seen from WSL.
var wslBrowsers = []string{
	"/mnt/c/Program Files (x86)/Microsoft/Edge/Application/msedge.exe",
	"/mnt/c/Program Files/Microsoft/Edge/Application/msedge.exe",
	"/mnt/c/Program Files/Google/Chrome/Application/chrome.exe",
	"/mnt/c/Program Files (x86)/Google/Chrome/Application/chrome.exe",
}

var windowsBrowsers = []struct{ dir, exe string }{
	{`Microsoft\Edge`, "msedge.exe"},
	{`Google\Chrome`, "chrome.exe"},
}

// OpenWindow opens url in an app window of an installed Chromium browser, or
// in the default browser, and returns without waiting for it: the browser runs
// detached and the cockpit never depends on it. The first candidate that
// starts wins; if none does, the error says why.
func OpenWindow(url string) error {
	launches := WindowLaunches(runtime.GOOS, os.Getenv, exec.LookPath, func(path string) bool {
		info, err := os.Stat(path)
		return err == nil && !info.IsDir()
	}, url)
	var failures []error
	for _, launch := range launches {
		cmd := exec.Command(launch.Name, launch.Args...)
		if err := cmd.Start(); err != nil {
			failures = append(failures, err)
			continue
		}
		// Not waited for: the cockpit must outlive the window and ignore its end.
		go func() { _ = cmd.Wait() }()
		return nil
	}
	if len(failures) == 0 {
		return errors.New("no browser found")
	}
	parts := make([]string, len(failures))
	for i, err := range failures {
		parts[i] = err.Error()
	}
	return fmt.Errorf("no browser started: %s", strings.Join(parts, "; "))
}
