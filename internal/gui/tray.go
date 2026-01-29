//go:build !windows

package gui

import (
	"fyne.io/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// initTray initializes the system tray icon and menu.
// It runs systray.Run in a separate goroutine.
func (a *App) initTray(icon []byte) {
	a.trayIcon = icon

	go systray.Run(func() {
		systray.SetIcon(icon)
		systray.SetTooltip("fxTunnel")

		mShow := systray.AddMenuItem("Show", "Show fxTunnel window")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Quit fxTunnel")

		go func() {
			for {
				select {
				case <-mShow.ClickedCh:
					if a.ctx != nil {
						runtime.WindowShow(a.ctx)
					}
				case <-mQuit.ClickedCh:
					systray.Quit()
					if a.ctx != nil {
						runtime.Quit(a.ctx)
					}
					return
				}
			}
		}()
	}, func() {
		// onExit - cleanup
	})
}

// cleanupTray stops the system tray.
func (a *App) cleanupTray() {
	systray.Quit()
}

// HasTray returns true if system tray is supported on this platform.
func (a *App) HasTray() bool {
	return true
}
