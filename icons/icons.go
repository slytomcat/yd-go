package icons

import (
	"context"
	"sync"
	"time"
)

const busyIconsCnt = 5

var interval = time.Millisecond * 333

// iconSet is set of one theme icons
type iconsSet struct {
	busyIcons [busyIconsCnt][]byte // busy icons set for icon animation for index and busy statuses
	idleIcon  []byte               // idle icon data
	pauseIcon []byte               // pause icon data
	errorIcon []byte               // error icon data
}

var (
	lightSet = &iconsSet{
		busyIcons: [busyIconsCnt][]byte{lightBusy1, lightBusy2, lightBusy3, lightBusy4, lightBusy5},
		idleIcon:  lightIdle,
		pauseIcon: lightPause,
		errorIcon: lightError,
	}
	darkSet = &iconsSet{
		busyIcons: [busyIconsCnt][]byte{darkBusy1, darkBusy2, darkBusy3, darkBusy4, darkBusy5},
		idleIcon:  darkIdle,
		pauseIcon: darkPause,
		errorIcon: darkError,
	}
)

// Icon is the icon helper
type Icon struct {
	*iconsSet                    // current theme icons set
	LogoIcon        []byte       // bytes of logo icon
	lock            sync.Mutex   // data protection lock
	currentStatus   string       // current icon status
	currentBusyIcon int          // current icon number for busy animation
	setFunc         func([]byte) // function to set icon
	ticker          *time.Ticker // ticker for icon animation
	stopper         func()       // helper stop function
}

// NewIcon initializes the icon helper, sets 'paused' icon ac initial and returns the helper.
// Use icon.Close() for properly utilization of the icon helper resources.
// The helper provides 'paused' 'idle' 'error' icons and animated 'busy' icon. In additional it provides LogoIcon.
// Icons 'paused' 'idle' 'error' and 'busy' are provided in one of two themes: "light" or "dark" for light or dark DE themes.
func NewIcon(theme string, setFunc func([]byte)) *Icon {
	ctx, cancel := context.WithCancel(context.Background())
	i := &Icon{
		currentStatus:   "paused",
		currentBusyIcon: 0,
		LogoIcon:        logo,
		setFunc:         setFunc,
		ticker:          time.NewTicker(time.Hour),
		stopper:         cancel,
	}
	i.ticker.Stop()
	i.SetTheme(theme)
	go i.loop(ctx)
	return i
}

// SetTheme select one of the icons' themes: "light" or "dark" and update icon from new theme via setFunc.
func (i *Icon) SetTheme(theme string) {
	i.lock.Lock()
	defer i.lock.Unlock()
	switch theme {
	case "light":
		i.iconsSet = lightSet
	case "dark":
		i.iconsSet = darkSet
	}
	i.setIcon()
}

// setIcon sets the current icon image via i.SetFunc
func (i *Icon) setIcon() {
	switch i.currentStatus {
	case "busy":
		i.setFunc(i.busyIcons[i.currentBusyIcon])
	case "idle":
		i.setFunc(i.idleIcon)
	case "paused":
		i.setFunc(i.pauseIcon)
	case "error":
		i.setFunc(i.errorIcon)
	}
}

// Set sets the icon for status "busy" (animated), "idle", "paused" and "error"
func (i *Icon) Set(status string) {
	i.lock.Lock()
	defer i.lock.Unlock()
	if i.currentStatus != "busy" && status == "busy" { // not busy -> busy
		i.ticker.Reset(interval)
	}
	if status != "busy" && i.currentStatus == "busy" { // busy -> not busy
		i.ticker.Stop()
	}
	i.currentStatus = status
	i.setIcon()
}

func (i *Icon) loop(ctx context.Context) {
	for {
		select {
		case <-i.ticker.C:
			i.lock.Lock()
			i.currentBusyIcon = (i.currentBusyIcon + 1) % busyIconsCnt
			i.setIcon()
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
