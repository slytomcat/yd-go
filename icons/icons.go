package icons

import (
	"context"
	"fmt"
	"sync"
	"time"
)

var interval = time.Millisecond * 333

// Icon is the icon helper
type Icon struct {
	NotifyIcon    []byte
	lock          sync.Mutex // data protection lock
	currentStatus string
	currentIcon   int
	busyIcons     [5][]byte
	idleIcon      []byte
	pauseIcon     []byte
	errorIcon     []byte
	setFunc       func([]byte)
	ticker        *time.Ticker
	stopper       func()
}

// NewIcon initializes the icon helper and returns it.
// Use icon.CleanUp() for properly utilization of icon helper.
func NewIcon(theme string, set func([]byte)) (*Icon, error) {
	ctx, cancel := context.WithCancel(context.Background())
	i := &Icon{
		currentStatus: "",
		currentIcon:   0,
		NotifyIcon:    yd128,
		setFunc:       set,
		ticker:        time.NewTicker(interval),
		stopper:       cancel,
	}
	i.ticker.Stop()
	if err := i.SetTheme(theme); err != nil {
		return nil, err
	}
	i.setFunc(i.pauseIcon)
	go i.loop(ctx)
	return i, nil
}

// SetTheme select one of the icons' themes
func (i *Icon) SetTheme(theme string) error {
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
	default:
		return fmt.Errorf("wrong theme name: '%s' (should be 'dark' or 'light')", theme)
	}
	if i.currentStatus != "" {
		i.setIcon(i.currentStatus)
	}
	return nil
}

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
			i.currentIcon = (i.currentIcon + 1) % 5
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
