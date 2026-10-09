package codexappserver

import (
	"context"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestAppServerArgsKeepDefaultAndScopeMXCToChildCommand(t *testing.T) {
	if got, want := appServerArgs(Config{}), []string{"app-server", "--listen", "stdio://"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("default args = %#v, want %#v", got, want)
	}
	got := appServerArgs(Config{WindowsSandboxBackend: WindowsSandboxBackendMXC})
	want := []string{"-c", "windows.sandbox=mxc", "app-server", "--listen", "stdio://"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MXC args = %#v, want global CLI override before app-server: %#v", got, want)
	}
	got = appServerArgs(Config{PermissionProfile: ":workspace", WindowsSandboxBackend: WindowsSandboxBackendMXC})
	want = []string{"-c", "approval_policy=never", "-c", "sandbox_workspace_write={writable_roots=[],network_access=false,exclude_tmpdir_env_var=false,exclude_slash_tmp=false}", "-c", "windows.sandbox=mxc", "app-server", "--listen", "stdio://"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("owned workspace args = %#v, want %#v", got, want)
	}
	if got := appServerArgs(Config{PermissionProfile: ":read-only"}); !reflect.DeepEqual(got, []string{"app-server", "--listen", "stdio://"}) {
		t.Fatalf("nondefault profile changed approval policy: %#v", got)
	}
}

func TestWindowsSandboxBackendFailsClosedOffWindowsBeforeProcessLaunch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows-only setting is supported on this platform")
	}
	if err := validateSandboxPlatform(Config{WindowsSandboxBackend: WindowsSandboxBackendMXC}); err == nil {
		t.Fatal("accepted the Windows-only sandbox backend on a non-Windows host")
	}
	if err := verifyVersion(context.Background(), Config{Command: "must-not-be-launched", WindowsSandboxBackend: WindowsSandboxBackendMXC}, nil); err == nil || !strings.Contains(err.Error(), "only supported on Windows") {
		t.Fatal("version verification should fail on platform before attempting to launch the command")
	}
}
