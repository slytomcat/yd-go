package notify

import (
	"sync/atomic"

	"github.com/godbus/dbus/v5"
)

// Notify holds D-Bus connection and defaults for the notifications.
type Notify struct {
	app     string
	icon    string
	replace bool
	time    int
	conn    *dbus.Conn
	connObj dbus.BusObject
	lastID  uint32
}

const (
	dBusDest = "org.freedesktop.Notifications"
	dBusPath = "/org/freedesktop/Notifications"
)

// New creates new Notify component.
// The application is the name of application.
// The defaultIcon is icon name/path to be used for notification when no icon specified during the Send call.
// True value of replace means that a new notification will replace the previous one if it is still displayed.
// The time sets the time in milliseconds after which the notification will disappear. Set it to -1 to use Desktop default settings.
func New(application, defaultIcon string, replace bool, time int) (*Notify, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	notify := &Notify{
		app:     application,
		icon:    defaultIcon,
		replace: replace,
		time:    time,
		conn:    conn,
		connObj: conn.Object(dBusDest, dBusPath),
	}
	if _, err = notify.Cap(); err != nil {
		return nil, err
	}
	return notify, nil
}

// Close closes d-bus connection. Call it on app exit or similar cases.
func (n *Notify) Close() {
	n.conn.Close()
}

// Send sends the desktop notification. If icon is not provided ("") then defaultIcon passed to New is used.
func (n *Notify) Send(icon, title, message string) {

	if icon == "" {
		icon = n.icon
	}
	var last uint32
	if n.replace {
		last = atomic.LoadUint32(&n.lastID)
	}
	call := n.connObj.Call(dBusDest+".Notify", dbus.Flags(0),
		n.app, last, icon, title, message, []string{}, map[string]interface{}{}, n.time)
	if call.Err == nil {
		atomic.StoreUint32(&n.lastID, call.Body[0].(uint32))
	}
	// ignore any error as it can be called even New return error.
}

// Cap returns the notification server capabilities
func (n *Notify) Cap() ([]string, error) {
	call := n.connObj.Call(dBusDest+".GetCapabilities", dbus.Flags(0))
	if call.Err != nil {
		return nil, call.Err
	}
	return call.Body[0].([]string), nil
}
