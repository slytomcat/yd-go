package icons

import (
	"path"
)

var (
	// IconBusy - set of 5 icons to be shown in busy status (with animation)
	IconBusy   [5]string 
	// IconError - is the icon to show error status
	IconError  string
	// IconIdle - is shown whe daemon do nothing (waits fo events)
	IconIdle   string
	// IconPause - is shown in inactive status (not started/paused)
	IconPause  string
	// IconNotify - 128x128 icon to show in notifications
	IconNotify string
)

// SetTheme sets the Icon* variable according to selected theme
func SetTheme(icoHome, theme string) {

	themePath := path.Join(icoHome, theme)

	IconNotify = path.Join(icoHome, "yd-128.png")

	IconBusy = [5]string{
		path.Join(themePath, "busy1.png"),
		path.Join(themePath, "busy2.png"),
		path.Join(themePath, "busy3.png"),
		path.Join(themePath, "busy4.png"),
		path.Join(themePath, "busy5.png"),
	}
	IconError = path.Join(themePath, "error.png")
	IconIdle = path.Join(themePath, "idle.png")
	IconPause = path.Join(themePath, "pause.png")
}
