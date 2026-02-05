package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/mephistofox/fxtunnel/internal/gui"
)

//go:embed all:gui/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

// Set via ldflags at build time
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// Setup logging
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}
	log := zerolog.New(output).With().Timestamp().Logger()

	// Create app
	app := gui.NewApp(log)
	app.SetIcon(appIcon)

	// Set build info
	app.SetBuildInfo(Version, BuildTime)

	// Re-create logger with GUI hook so all logs appear in the Logs view
	log = log.Hook(app.LogHook())
	app.UpdateLogger(log)

	// Run Wails application
	err := wails.Run(&options.App{
		Title:     "fxTunnel",
		Width:     1024,
		Height:    768,
		MinWidth:  800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup:        app.Startup,
		OnShutdown:       app.Shutdown,
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			if app.HasTray() && app.SettingsService != nil && app.SettingsService.GetMinimizeToTray() {
				wailsRuntime.WindowHide(ctx)
				return true
			}
			return false
		},
		Bind: []interface{}{
			app,
			app.TunnelService,
			app.AuthService,
			app.BundleService,
			app.SettingsService,
			app.HistoryService,
			app.DomainService,
			app.CustomDomainService,
			app.SyncService,
			app.InspectService,
			app.UpdateService,
			app.AccountService,
		},
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  false,
				HideTitleBar:               false,
				FullSizeContent:            true,
				UseToolbar:                 false,
				HideToolbarSeparator:       true,
			},
			About: &mac.AboutInfo{
				Title:   "fxTunnel",
				Message: "Reverse tunneling client",
			},
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
		Linux: &linux.Options{
			WindowIsTranslucent: false,
			Icon:                appIcon,
			ProgramName:         "fxtunnel-gui",
		},
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
