package icons

import (
	"os"

	"github.com/slytomcat/llog"
)

var (
	// BusyIcons icons array to show bysy status
	BusyIcons [5][]byte
	// IdleIcon icon to show idle status
	IdleIcon []byte
	// PauseIcon icon to show paused status
	PauseIcon []byte
	// ErrorIcon icon to show error status
	ErrorIcon []byte
	// NotifyIcon is a path to icon to show in notifications
	NotifyIcon string
)

func init() {
	file, err := os.CreateTemp(os.TempDir(), "yd_notify_icon*.png")
	if err != nil {
		panic(err)
	}
	_, err = file.Write(yd128)
	if err != nil {
		panic(err)
	}
	NotifyIcon = file.Name()
}

// SelectTheme select one of the icons' themes
func SelectTheme(theme string) {
	switch theme {
	case "light":
		BusyIcons = [5][]byte{lightBusy1, lightBusy2, lightBusy3, lightBusy4, lightBusy5}
		IdleIcon = lightIdle
		PauseIcon = lightPause
		ErrorIcon = lightError
	case "dark":
		BusyIcons = [5][]byte{darkBusy1, darkBusy2, darkBusy3, darkBusy4, darkBusy5}
		IdleIcon = darkIdle
		PauseIcon = darkPause
		ErrorIcon = darkError
	default:
		llog.Criticalf("wrong theme: '%s' (should be 'dark' or 'light')", theme)
	}
}

// CleanUp removes temporary file for notification icon
func CleanUp() {
	if err := os.Remove(NotifyIcon); err != nil {
		panic(err)
	}
}
