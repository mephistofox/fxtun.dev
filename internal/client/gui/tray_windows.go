package gui

// initTray is a no-op on Windows.
// fyne.io/systray conflicts with Wails v2 on Windows due to
// runtime.LockOSThread() in systray's init(), causing the window
// to close unexpectedly after certain operations.
func (a *App) initTray(_ []byte) {}

// cleanupTray is a no-op on Windows.
func (a *App) cleanupTray() {}

// HasTray returns false on Windows where systray is disabled.
func (a *App) HasTray() bool {
	return false
}
