package cockpit

import (
	"errors"
	"reflect"
	"testing"
)

func TestWindowLaunchesChooseTheAppWindowOfAnInstalledChromiumBrowser(t *testing.T) {
	const url = "http://127.0.0.1:7341/"
	const app = "--app=" + url
	edge := `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`
	chrome := `C:\Program Files\Google\Chrome\Application\chrome.exe`
	env := func(values map[string]string) func(string) string {
		return func(key string) string { return values[key] }
	}
	path := func(found map[string]string) func(string) (string, error) {
		return func(name string) (string, error) {
			if p, ok := found[name]; ok {
				return p, nil
			}
			return "", errors.New("not found")
		}
	}
	files := func(present ...string) func(string) bool {
		return func(p string) bool {
			for _, f := range present {
				if f == p {
					return true
				}
			}
			return false
		}
	}
	windowsEnv := env(map[string]string{"ProgramFiles": `C:\Program Files`, "ProgramFiles(x86)": `C:\Program Files (x86)`, "LocalAppData": `C:\Users\o\AppData\Local`})
	fallbackWindows := Launch{"rundll32", []string{"url.dll,FileProtocolHandler", url}}
	tests := []struct {
		name   string
		goos   string
		env    func(string) string
		path   func(string) (string, error)
		exists func(string) bool
		want   []Launch
	}{
		{"windows prefers Edge, then Chrome, then the default browser", "windows", windowsEnv, path(nil), files(edge, chrome),
			[]Launch{{edge, []string{app}}, {chrome, []string{app}}, fallbackWindows}},
		{"windows finds Edge on the path", "windows", env(nil), path(map[string]string{"msedge.exe": `D:\edge\msedge.exe`}), files(),
			[]Launch{{`D:\edge\msedge.exe`, []string{app}}, fallbackWindows}},
		{"windows without a browser uses the default browser", "windows", windowsEnv, path(nil), files(), []Launch{fallbackWindows}},
		{"macOS opens Chrome as an app window", "darwin", env(map[string]string{"HOME": "/Users/o"}), path(nil), files("/Applications/Google Chrome.app", "/Users/o/Applications/Microsoft Edge.app"),
			[]Launch{{"open", []string{"-na", "Google Chrome", "--args", app}}, {"open", []string{"-na", "Microsoft Edge", "--args", app}}, {"open", []string{url}}}},
		{"macOS without Chromium uses open", "darwin", env(nil), path(nil), files(), []Launch{{"open", []string{url}}}},
		{"linux prefers an installed Chromium browser", "linux", env(nil), path(map[string]string{"chromium": "/usr/bin/chromium", "microsoft-edge": "/usr/bin/microsoft-edge", "xdg-open": "/usr/bin/xdg-open"}), files(),
			[]Launch{{"/usr/bin/chromium", []string{app}}, {"/usr/bin/microsoft-edge", []string{app}}, {"/usr/bin/xdg-open", []string{url}}}},
		{"linux without a browser or opener has nothing to start", "linux", env(nil), path(nil), files(), nil},
		{"linux falls back to xdg-open", "linux", env(nil), path(map[string]string{"xdg-open": "/usr/bin/xdg-open"}), files(),
			[]Launch{{"/usr/bin/xdg-open", []string{url}}}},
		{"WSL opens on the Windows side first", "linux", env(map[string]string{"WSL_DISTRO_NAME": "Ubuntu"}), path(map[string]string{"cmd.exe": "/mnt/c/Windows/system32/cmd.exe", "xdg-open": "/usr/bin/xdg-open"}), files(),
			[]Launch{
				{"/mnt/c/Windows/system32/cmd.exe", []string{"/c", "start", "", "msedge", app}},
				{"/mnt/c/Windows/system32/cmd.exe", []string{"/c", "start", "", "chrome", app}},
				{"/mnt/c/Windows/system32/cmd.exe", []string{"/c", "start", "", url}},
				{"/usr/bin/xdg-open", []string{url}},
			}},
		{"WSL without Windows interop uses the Linux candidates", "linux", env(map[string]string{"WSL_DISTRO_NAME": "Ubuntu"}), path(map[string]string{"xdg-open": "/usr/bin/xdg-open"}), files(),
			[]Launch{{"/usr/bin/xdg-open", []string{url}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WindowLaunches(tt.goos, tt.env, tt.path, tt.exists, url)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %+v\nwant %+v", got, tt.want)
			}
		})
	}
}
