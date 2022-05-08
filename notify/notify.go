package notify

import (
	"github.com/godbus/dbus/v5"
)

// Notify holds D-Bus connection
type Notify struct {
	app     string
	icon    string
	replace bool
	time    int
	conn    *dbus.Conn
	lastID  uint32
}

// New creates new Notify component
func New(app, defailtIcon string, replace bool, time int) (*Notify, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	return &Notify{
		app:     app,
		icon:    defailtIcon,
		replace: replace,
		time:    time,
		conn:    conn,
	}, nil
}

// Send sends the desktop notification
func (n *Notify) Send(icon, title, message string) {
	if icon == "" {
		icon = n.icon
	}
	var last uint32
	if n.replace {
		last = n.lastID
	}
	obj := n.conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
	call := obj.Call("org.freedesktop.Notifications.Notify", dbus.Flags(0),
		n.app, last, icon, title, message, []string{}, map[string]interface{}{}, n.time)
	if call.Err != nil {
		panic(call.Err)
	}
	n.lastID = call.Body[0].(uint32)
}

// Cap returns the notification server capabilities
func (n *Notify) Cap() []string {
	obj := n.conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
	call := obj.Call("org.freedesktop.Notifications.GetCapabilities", dbus.Flags(0))
	if call.Err != nil {
		panic(call.Err)
	}
	return call.Body[0].([]string)
}
