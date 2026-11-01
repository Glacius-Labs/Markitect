//go:build native

// Command markitect-desktop is Markitect Desktop as a native app: the
// service of internal/bootstrap shown in windows of the platform's own web
// view, with a tray icon and the platform's folder dialog. It builds only with
// the native tag, so that the rest of the repository needs no C toolchain or
// GTK; on Windows it is built as a GUI program, so that it opens no terminal:
//
//	go build -tags native -ldflags=-H=windowsgui ./cmd/markitect-desktop
package main

import (
	"context"
	_ "embed"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/Glacius-Labs/Markitect/internal/bootstrap"
)

// icon is the app's icon: windows, taskbar, Dock and tray.
//
//go:embed icon.png
var icon []byte

// trayTemplate is the tray icon on macOS, whose menu bar colours a
// template's shape itself.
//
//go:embed tray-template.png
var trayTemplate []byte

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dir, dirErr := bootstrap.DesktopFolder()
	app := application.New(application.Options{
		Name:        "Markitect",
		Description: "Model-first development with delegated realization",
		Icon:        icon,
		// Closing the last window stops nothing; the tray opens another.
		Windows: application.WindowsOptions{
			DisableQuitOnLastWindowClosed: true,
			WebviewUserDataPath:           filepath.Join(dir, "WebView"),
		},
		Linux: application.LinuxOptions{DisableQuitOnLastWindowClosed: true},
	})
	s := &shell{app: app, dir: dir, started: make(chan struct{})}
	if dirErr != nil {
		s.failStart(dirErr)
		os.Exit(1)
	}
	// Before the service starts its first program. Without it every Git call
	// would flash a console window over the app, so the app does not start.
	if err := hideChildConsoles(); err != nil {
		s.failStart(err)
		os.Exit(1)
	}
	svc, err := bootstrap.ListenDesktop(ctx, bootstrap.DesktopWindows{Open: s.open, ChooseFolder: s.chooseFolder})
	switch {
	case err != nil:
		s.failStart(err)
		os.Exit(1)
	case svc != nil:
		if err := s.run(ctx, cancel, svc); err != nil {
			svc.Close()
			s.record(err)
			os.Exit(1)
		}
	}
	// Otherwise this user's Markitect already runs and opened the window.
}

// shell is the native app around the service: its windows, tray and
// dialogs. The page in a window talks only to the service; the shell binds
// no Go to it.
type shell struct {
	app     *application.App
	dir     string        // the config folder, where a failure without a window is recorded
	started chan struct{} // closed once the app runs
}

// run serves once the app runs, and keeps both until the user quits from the
// tray, or from the app menu on macOS.
func (s *shell) run(ctx context.Context, cancel context.CancelFunc, svc *bootstrap.DesktopService) error {
	s.tray(svc)
	s.app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		close(s.started)
		go func() {
			if err := svc.Serve(ctx); err != nil {
				s.fail(err)
			}
		}()
	})
	// On macOS a click on the Dock icon with no window open asks for one.
	s.app.Event.OnApplicationEvent(events.Mac.ApplicationShouldHandleReopen, func(*application.ApplicationEvent) { s.show(svc) })
	// Shutdown runs on the main thread, which a request in flight may wait
	// for, such as an open folder dialog. So it waits for nothing that needs
	// that thread: closing removes the record first, and the process ends
	// after a moment either way.
	s.app.OnShutdown(func() {
		cancel()
		closed := make(chan struct{})
		go func() {
			svc.Close()
			close(closed)
		}()
		select {
		case <-closed:
		case <-time.After(time.Second):
		}
	})
	return s.app.Run()
}

func (s *shell) tray(svc *bootstrap.DesktopService) {
	tray := s.app.SystemTray.New()
	tray.SetTooltip("Markitect")
	menu := s.app.NewMenu()
	menu.Add("Open Markitect").OnClick(func(*application.Context) { s.show(svc) })
	menu.Add("New window").OnClick(func(*application.Context) { s.newWindow(svc) })
	menu.AddSeparator()
	menu.Add("Quit Markitect").OnClick(func(*application.Context) { s.app.Quit() })
	tray.SetMenu(menu)
	if runtime.GOOS == "darwin" {
		// A click on a menu bar icon opens its menu.
		tray.SetTemplateIcon(trayTemplate)
		return
	}
	tray.SetIcon(icon)
	tray.OnClick(func() { s.show(svc) })
}

// open opens a window on url, a link with a launch code that the service
// made; the service calls it for every window.
func (s *shell) open(url string) error {
	<-s.started
	s.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Markitect",
		Width:            1280,
		Height:           820,
		MinWidth:         390,
		MinHeight:        560,
		BackgroundColour: application.NewRGB(0x21, 0x21, 0x21), // the design system's dark background
		URL:              url,
	})
	return nil
}

// show brings a window to the front, or has the service open one when none
// is open.
func (s *shell) show(svc *bootstrap.DesktopService) {
	w := s.app.Window.Current()
	if w == nil {
		if all := s.app.Window.GetAll(); len(all) > 0 {
			w = all[0]
		}
	}
	if w == nil {
		s.newWindow(svc)
		return
	}
	w.UnMinimise()
	w.Show()
	w.Focus()
}

func (s *shell) newWindow(svc *bootstrap.DesktopService) {
	if err := svc.Window(); err != nil {
		s.alert(err, nil)
	}
}

// chooseFolder asks for a folder in the platform's dialog, over the window
// in front.
func (s *shell) chooseFolder(_ context.Context, start string) (string, error) {
	d := s.app.Dialog.OpenFile().CanChooseFiles(false).CanChooseDirectories(true).CanCreateDirectories(true).SetTitle("Choose a folder")
	if start != "" {
		d.SetDirectory(start)
	}
	if w := s.app.Window.Current(); w != nil {
		d.AttachToWindow(w)
	}
	path, err := d.PromptForSingleSelection()
	if err != nil && err.Error() == "cancelled by user" {
		return "", nil // Windows reports a cancel as this error
	}
	return path, err
}

// alert shows err in a dialog, and runs then once the user dismissed it.
func (s *shell) alert(err error, then func()) {
	d := s.app.Dialog.Error().SetTitle("Markitect").SetMessage(err.Error())
	ok := d.AddButton("OK")
	if then != nil {
		ok.OnClick(then)
	}
	d.SetDefaultButton(ok).SetCancelButton(ok).Show()
}

// fail shows err and ends the app.
func (s *shell) fail(err error) { s.alert(err, s.app.Quit) }

// failStart shows err when the service could not start; the app runs only
// for the dialog. When not even the app runs, the error is recorded.
func (s *shell) failStart(err error) {
	s.app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		go s.fail(err)
	})
	if runErr := s.app.Run(); runErr != nil {
		s.record(errors.Join(err, runErr))
	}
}

// record writes a failure that no window can show, such as a missing web
// view, to desktop-error.log in the config folder, since the app has no
// terminal.
func (s *shell) record(err error) {
	if s.dir == "" || os.MkdirAll(s.dir, 0o700) != nil {
		return
	}
	line := time.Now().Format(time.RFC3339) + " " + err.Error() + "\n"
	if f, openErr := os.OpenFile(filepath.Join(s.dir, "desktop-error.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); openErr == nil {
		_, _ = f.WriteString(line)
		_ = f.Close()
	}
}
