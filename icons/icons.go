package icons

import (
	"os"
	"sync/atomic"
	"time"

	"github.com/slytomcat/llog"
)

var interval = time.Millisecond * 333

// Icon is the icon helper
type Icon struct {
	NotifyIcon    string // path to notification icon stored as file on disk
	currentStatus string
	currentIcon   int64
	busyIcons     [5][]byte
	idleIcon      []byte
	pauseIcon     []byte
	errorIcon     []byte
	setFunc       func([]byte)
	ticker        *time.Ticker
	exit          chan struct{}
}

// NewIcon initializes the icon helper and retuns it.
// Use icon.CleanUp() for properly utilization of icon helper.
func NewIcon(theme string, set func([]byte)) *Icon {
	file, err := os.CreateTemp(os.TempDir(), "yd_notify_icon*.png")
	if err != nil {
		llog.Critical(err)
	}
	_, err = file.Write(yd128)
	if err != nil {
		llog.Critical(err)
	}

	i := &Icon{
		currentStatus: "",
		currentIcon:   0,
		NotifyIcon:    file.Name(),
		setFunc:       set,
		ticker:        time.NewTicker(interval),
		exit:          make(chan struct{}, 1),
	}
	i.ticker.Stop()
	i.SetTheme(theme)
	go i.loop()
	i.setFunc(i.pauseIcon)
	return i
}

// SetTheme select one of the icons' themes
func (i *Icon) SetTheme(theme string) {
	switch theme {
	case "light":
		i.busyIcons = [5][]byte{lightBusy1, lightBusy2, lightBusy3, lightBusy4, lightBusy5}
		i.idleIcon = lightIdle
		i.pauseIcon = lightPause
		i.errorIcon = lightError
	case "dark":
		i.busyIcons = [5][]byte{darkBusy1, darkBusy2, darkBusy3, darkBusy4, darkBusy5}
		i.idleIcon = darkIdle
		i.pauseIcon = darkPause
		i.errorIcon = darkError
	default:
		llog.Criticalf("wrong theme: '%s' (should be 'dark' or 'light')", theme)
	}
	if i.currentStatus != "" {
		i.setIcon(i.currentStatus)
	}
}

func (i *Icon) setIcon(status string) {
	switch status {
	case "busy", "index":
		i.setFunc(i.busyIcons[atomic.LoadInt64(&i.currentIcon)])
	case "idle":
		i.setFunc(i.idleIcon)
	case "none", "paused":
		i.setFunc(i.pauseIcon)
	default:
		i.setFunc(i.errorIcon)
	}
}

// Set sets the icon by status
func (i *Icon) Set(status string) {
	i.setIcon(status)
	if status == "busy" || status == "index" {
		if i.currentStatus != "busy" && i.currentStatus != "index" {
			i.ticker.Reset(interval)
		}
	} else {
		i.ticker.Stop()
	}
	i.currentStatus = status
}

func (i *Icon) loop() {
	for {
		select {
		case <-i.ticker.C:
			atomic.StoreInt64(&i.currentIcon, (i.currentIcon+1)%5)
			i.setFunc(i.busyIcons[i.currentIcon])
		case <-i.exit:
			return
		}
	}
}

// CleanUp removes temporary file for notification icon and stops internal loop
func (i *Icon) CleanUp() {
	if err := os.Remove(i.NotifyIcon); err != nil {
		llog.Critical(err)
	}
	i.ticker.Stop()
	close(i.exit)
}
