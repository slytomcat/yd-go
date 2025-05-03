package icons

import (
	"context"
	"sync"
	"time"
)

var interval = time.Millisecond * 333

// Icon is the icon helper
type Icon struct {
	NotifyIcon    []byte       // bytes of icon for notifications (png of ico)
	lock          sync.Mutex   // data protection lock
	currentStatus string       // current icon status
	currentIcon   int          // current icon number for busy animation
	busyIcons     [5][]byte    // busy icons set for icon animation
	idleIcon      []byte       // idle icon data
	pauseIcon     []byte       // pause icon data
	errorIcon     []byte       // error icon data
	setFunc       func([]byte) // function to set icon
	ticker        *time.Ticker // ticker for icon animation
	stopper       func()       // stop function
}

// NewIcon initializes the icon helper and returns it.
// Use icon.Close() for properly utilization of icon helper.
func NewIcon(theme string, setFunc func([]byte)) *Icon {
	ctx, cancel := context.WithCancel(context.Background())
	i := &Icon{
		currentStatus: "",
		currentIcon:   0,
		NotifyIcon:    yd128,
		setFunc:       setFunc,
		ticker:        time.NewTicker(time.Hour),
		stopper:       cancel,
	}
	i.ticker.Stop()
	i.SetTheme(theme)
	i.setFunc(i.pauseIcon)
	go i.loop(ctx)
	return i
}

// SetTheme select one of the icons' themes. The theme name must be "light" or "dark".
func (i *Icon) SetTheme(theme string) {
	i.lock.Lock()
	defer i.lock.Unlock()
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
	}
	if i.currentStatus != "" {
		i.setIcon(i.currentStatus)
	}
}

// setIcon sets the current icon image via i.SetFunc
func (i *Icon) setIcon(status string) {
	switch status {
	case "busy", "index":
		i.setFunc(i.busyIcons[i.currentIcon])
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
	i.lock.Lock()
	defer i.lock.Unlock()
	if status == "busy" || status == "index" {
		if i.currentStatus != "busy" && i.currentStatus != "index" {
			i.ticker.Reset(interval)
		}
	} else {
		i.ticker.Stop()
	}
	i.currentStatus = status
	i.setIcon(status)
}

func (i *Icon) loop(ctx context.Context) {
	for {
		select {
		case <-i.ticker.C:
			i.lock.Lock()
			i.currentIcon = (i.currentIcon + 1) % len(i.busyIcons)
			i.setFunc(i.busyIcons[i.currentIcon])
			i.lock.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

// Close stops internal loop
func (i *Icon) Close() {
	i.ticker.Stop()
	i.stopper()
}
