package notify

import (
	"bytes"
	"context"
	"image"
	_ "image/png"
	"sync/atomic"

	"github.com/godbus/dbus/v5"
)

// Notify holds D-Bus connection and defaults for the notifications.
type Notify struct {
	ctx       context.Context
	cancel    func()
	app       string
	iconHints map[string]dbus.Variant
	replace   bool
	time      int
	conn      *dbus.Conn
	connObj   dbus.BusObject
	lastID    atomic.Uint32
}

const (
	dBusDest = "org.freedesktop.Notifications"
	dBusPath = "/org/freedesktop/Notifications"
)

// New creates new Notify component.
// The application is the name of application.
// The icon is png/ico image data to use in Send as notify message icon.
// True value of replace means that a new notification will replace the previous one if it is still displayed.
// The time sets the time in milliseconds after which the notification will disappear. Set it to -1 to use Desktop default settings.
// It returns error in cases of D-BUS connection error or error of getting the notification server capabilities.
func New(application string, icon []byte, replace bool, time int) (*Notify, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	notify := &Notify{
		ctx:     ctx,
		cancel:  cancel,
		app:     application,
		replace: replace,
		time:    time,
		conn:    conn,
		connObj: conn.Object(dBusDest, dBusPath),
	}
	if _, err = notify.Cap(); err != nil {
		return nil, err
	}
	// perform heavy staff only when service is available
	notify.iconHints = map[string]dbus.Variant{"image-data": dbus.MakeVariant(convertToPixels(icon))}
	return notify, nil
}

// Close closes D-BUS connection. Call it on app exit or similar cases.
func (n *Notify) Close() {
	n.cancel()
	n.conn.Close()
}

// Send sends the desktop notification.
func (n *Notify) Send(title, message string) {
	var last uint32
	if n.replace {
		last = n.lastID.Load()
	}
	call := n.connObj.CallWithContext(n.ctx, dBusDest+".Notify", dbus.Flags(0), n.app, last, "", title, message, []string{}, n.iconHints, n.time)
	if call.Err == nil && n.replace {
		n.lastID.Store(call.Body[0].(uint32))
	}
	// ignore the rest possible errors
}

// ImageData is struct to hold image data into pix-buffer format as it declared into D-BUS specs.
type ImageData struct {
	Width, Height, RowStride int32
	HasAlpha                 bool
	BitsPerSample, Channels  int32
	ImageData                []byte
}

// convertToPixels is used to convert png/ico format into pix-buffer as it declared into D-BUS specs.
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
	call := n.connObj.CallWithContext(n.ctx, dBusDest+".GetCapabilities", dbus.Flags(0))
	if call.Err != nil {
		return nil, call.Err
	}
	return call.Body[0].([]string), nil
}
