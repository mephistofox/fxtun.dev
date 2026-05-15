package gui

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// logHook is a zerolog Hook that forwards log messages to the GUI frontend.
type logHook struct {
	app *App
}

// Run implements zerolog.Hook.
func (h *logHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if h.app.ctx == nil {
		return
	}

	// Only forward info, warnings and errors to the GUI
	if level < zerolog.InfoLevel {
		return
	}

	var levelStr string
	switch level {
	case zerolog.InfoLevel:
		levelStr = "info"
	case zerolog.WarnLevel:
		levelStr = "warn"
	case zerolog.ErrorLevel, zerolog.FatalLevel, zerolog.PanicLevel:
		levelStr = "error"
	default:
		levelStr = "warn"
	}

	runtime.EventsEmit(h.app.ctx, "log", map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     levelStr,
		"message":   msg,
	})
}
