package notify

import (
	"bytes"
	"image"
	_ "image/png"
	"sync/atomic"

	"github.com/godbus/dbus/v5"
)

// Notify holds D-Bus connection and defaults for the notifications.
type Notify struct {
	app       string
	iconHints map[string]dbus.Variant
	replace   bool
	time      int
	conn      *dbus.Conn
	connObj   dbus.BusObject
	lastID    uint32
}

const (
	dBusDest = "org.freedesktop.Notifications"
	dBusPath = "/org/freedesktop/Notifications"
)

// New creates new Notify component.
// The application is the name of application.
// The icon is png/ico image data to use in Send.
// True value of replace means that a new notification will replace the previous one if it is still displayed.
// The time sets the time in milliseconds after which the notification will disappear. Set it to -1 to use Desktop default settings.
func New(application string, icon []byte, replace bool, time int) (*Notify, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	notify := &Notify{
		app:       application,
		iconHints: map[string]dbus.Variant{"image-data": dbus.MakeVariant(convertToPixels(icon))},
		replace:   replace,
		time:      time,
		conn:      conn,
		connObj:   conn.Object(dBusDest, dBusPath),
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

// Send sends the desktop notification.
func (n *Notify) Send(title, message string) {
	var last uint32
	if n.replace {
		last = atomic.LoadUint32(&n.lastID)
	}
	call := n.connObj.Call(dBusDest+".Notify", dbus.Flags(0), n.app, last, "", title, message, []string{}, n.iconHints, n.time)
	if call.Err == nil {
		atomic.StoreUint32(&n.lastID, call.Body[0].(uint32))
	}
	// ignore rest possible errors
}

type ImageData struct {
	Width, Height, RowStride int32
	HasAlpha                 bool
	BitsPerSample, Channels  int32
	ImageData                []byte
}

func convertToPixels(data []byte) ImageData {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		panic(err)
	}
	img := image.NewRGBA(src.Bounds())
	for x := range src.Bounds().Dx() {
		for y := range src.Bounds().Dy() {
			img.Set(x, y, src.At(x, y))
		}
	}
	return ImageData{
		Width:         int32(img.Bounds().Dx()),
		Height:        int32(img.Bounds().Dy()),
		RowStride:     int32(img.Stride),
		HasAlpha:      true,
		BitsPerSample: 8,
		Channels:      4,
		ImageData:     img.Pix,
	}
}

// Cap returns the notification server capabilities
func (n *Notify) Cap() ([]string, error) {
	call := n.connObj.Call(dBusDest+".GetCapabilities", dbus.Flags(0))
	if call.Err != nil {
		return nil, call.Err
	}
	return call.Body[0].([]string), nil
}
