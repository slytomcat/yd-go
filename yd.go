// Copyleft 2017-2021 Sly_tom_cat (slytomcat@mail.ru)
// License: GPL v.3

//go:generate gotext -srclang=en update -out=catalog.go -lang=en,ru

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/getlantern/systray"
	"github.com/slytomcat/llog"
	"github.com/slytomcat/yd-go/icons"
	"github.com/slytomcat/yd-go/tools"
	"github.com/slytomcat/ydisk"
	"golang.org/x/text/message"
)

var (
	version = "local build"
	// msg is the Localization printer
	msg      *message.Printer
	statusTr map[string]string
)

const about = `yd-go is the GTK-based panel indicator for Yandex.Disk daemon.

	Version: %s

Copyleft 2017-%s Sly_tom_cat (slytomcat@mail.ru)

	License: GPL v.3

`

func notifySend(icon, title, body string) {
	llog.Debug("Message:", title, ":", body)
	err := exec.Command("notify-send", "-i", icon, title, body).Start()
	if err != nil {
		llog.Error(err)
	}
}

// LastT type is just map[strig]string protected by RWMutex to be read and set
// form different goroutines simultaneously
type LastT struct {
	m map[string]*string
	l sync.RWMutex
}

func newLastT() *LastT {
	return &LastT{
		m: map[string]*string{},
		l: sync.RWMutex{},
	}
}

func (l *LastT) set(key, value string) {
	l.l.Lock()
	l.m[key] = &value
	l.l.Unlock()
}

func (l *LastT) get(key string) string {
	l.l.RLock()
	defer l.l.RUnlock()
	return *l.m[key]
}

func (l *LastT) len() int {
	l.l.RLock()
	defer l.l.RUnlock()
	return len(l.m)
}

type menu struct {
	status     *systray.MenuItem
	size1      *systray.MenuItem
	size2      *systray.MenuItem
	last       *systray.MenuItem
	lasts      []*systray.MenuItem
	lastCancel func()
	start      *systray.MenuItem
	stop       *systray.MenuItem
	out        *systray.MenuItem
	path       *systray.MenuItem
	site       *systray.MenuItem
	help       *systray.MenuItem
	about      *systray.MenuItem
	don        *systray.MenuItem
	quit       *systray.MenuItem
}

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	// Initialize application and receive the application configuration
	AppCfg := tools.AppInit("yd-go")

	// Initialize translations
	lng := os.Getenv("LANG")
	if len(lng) > 2 {
		lng = lng[:2]
	}

	llog.Infof("Local language is: %v", lng)
	msg = message.NewPrinter(message.MatchLanguage(lng))

	// Create new ydisk interface
	YD, err := ydisk.NewYDisk(AppCfg["Conf"].(string))
	if err != nil {
		llog.Critical("Fatal error:", err)
	}
	var ok bool
	// Start daemon if it is configured
	if start, ok := AppCfg["StartDaemon"].(bool); start {
		go YD.Start()
	} else if !ok {
		llog.Critical("Config read error: StartDaemon should be bool")
	}
	// Read stop flag (to stop the daemon on exit)
	stop, ok := AppCfg["StopDaemon"].(bool)
	if !ok {
		llog.Critical("Config read error: StopDaemon should be bool")
	}
	note, ok := AppCfg["Notifications"].(bool)
	if !ok {
		llog.Critical("Config error:", err)
	}

	// Initialize icon theme
	var theme string
	if theme, ok = AppCfg["Theme"].(string); !ok {
		llog.Critical("Config read error: Theme should be string")
	}
	icons.SelectTheme(theme)
	// Initialize systray icon
	systray.SetIcon(icons.PauseIcon)

	// Initialize status localization
	statusTr = map[string]string{
		"idle":   msg.Sprintf("idle"),
		"index":  msg.Sprintf("index"),
		"busy":   msg.Sprintf("busy"),
		"none":   msg.Sprintf("none"),
		"paused": msg.Sprintf("paused"),
	}

	m := new(menu)
	systray.SetTitle("yd-go indicator")
	// Initialize systray menu
	m.status = systray.AddMenuItem("", "")
	m.status.Disable()
	m.size1 = systray.AddMenuItem("", "")
	m.size1.Disable()
	m.size2 = systray.AddMenuItem("", "")
	m.size2.Disable()
	systray.AddSeparator()
	m.last = systray.AddMenuItem(msg.Sprintf("Last synchronized"), "")
	m.lasts = make([]*systray.MenuItem, 10)
	for i := 0; i < 10; i++ {
		m.lasts[i] = m.last.AddSubMenuItem("", "")
		m.lasts[i].Hide()
	}
	m.last.Disable()
	m.lastCancel = nil
	systray.AddSeparator()
	m.start = systray.AddMenuItem(msg.Sprintf("Start daemon"), "")
	m.start.Hide()
	m.stop = systray.AddMenuItem(msg.Sprintf("Stop daemon"), "")
	m.stop.Hide()
	systray.AddSeparator()
	m.out = systray.AddMenuItem(msg.Sprintf("Show daemon output"), "")
	m.path = systray.AddMenuItem(msg.Sprintf("Open: %s", YD.Path), "")
	m.site = systray.AddMenuItem(msg.Sprintf("Open YandexDisk in browser"), "")
	systray.AddSeparator()
	m.help = systray.AddMenuItem(msg.Sprintf("Help"), "")
	m.about = systray.AddMenuItem(msg.Sprintf("About"), "")
	m.don = systray.AddMenuItem(msg.Sprintf("Donations"), "")
	systray.AddSeparator()
	m.quit = systray.AddMenuItem(msg.Sprintf("Quit"), "")

	// Start handlers
	go menuHandler(YD, m, stop)   // handler for GUI events
	go changeHandler(YD, m, note) // handler for YDisk events
}

func menuHandler(YD *ydisk.YDisk, m *menu, stop bool) {
	llog.Debug("Menu handler started")
	defer func() {
		llog.Debug("Menu handler exited.")
		YD.Close() // it closes Changes channel -> closed channel closes disk event handler
	}()

	for {
		select {
		case <-m.start.ClickedCh:
			m.start.Disable()
			go YD.Start()
		case <-m.stop.ClickedCh:
			m.stop.Disable()
			go YD.Stop()
		case <-m.out.ClickedCh:
			notifySend(icons.IconNotify, msg.Sprintf("Yandex.Disk daemon output"), YD.Output())
		case <-m.path.ClickedCh:
			tools.XdgOpen(YD.Path)
		case <-m.site.ClickedCh:
			tools.XdgOpen("https://disk.yandex.com")
		case <-m.help.ClickedCh:
			tools.XdgOpen("https://github.com/slytomcat/YD.go/wiki/FAQ&SUPPORT")
		case <-m.about.ClickedCh:
			notifySend(icons.IconNotify, " ", msg.Sprintf(about, version, time.Now().Format("2006")))
		case <-m.don.ClickedCh:
			tools.XdgOpen("https://github.com/slytomcat/yd-go/wiki/Donations")
		case <-m.quit.ClickedCh:
			llog.Debug("Exit requested.")
			// Stop daemon if it is configured
			if stop {
				YD.Stop()
			}
			return
		}
	}
}

func changeHandler(YD *ydisk.YDisk, m *menu, note bool) {
	defer systray.Quit() // request for exit from systray main loop (gtk.main())
	llog.Debug("Changes handler started")
	defer llog.Debug("Changes handler exited.")
	// Prepare the staff for icon animation
	currentIcon := 0
	tick := time.NewTimer(333 * time.Millisecond)
	defer tick.Stop()
	currentStatus := ""
	for {
		select {
		case <-tick.C: //  Icon timer event
			currentIcon = (currentIcon + 1) % 5
			if currentStatus == "busy" || currentStatus == "index" {
				systray.SetIcon(icons.BusyIcons[currentIcon])
				tick.Reset(333 * time.Millisecond)
			}
		case yds, ok := <-YD.Changes: // get YDisk change event
			if !ok { // as Changes channel closed - exit
				return
			}
			currentStatus = yds.Stat
			st := strings.Join([]string{statusTr[yds.Stat], yds.Prog, yds.Err, tools.ShortName(yds.ErrP, 30)}, " ")
			m.status.SetTitle(msg.Sprintf("Status: %s", st))
			if yds.Stat == "error" {
				m.status.SetTooltip(fmt.Sprintf("%s\nPath: %s", yds.Err, yds.ErrP))
			} else {
				m.status.SetTooltip("")
			}
			m.size1.SetTitle(msg.Sprintf("Used: %s/%s", yds.Used, yds.Total))
			m.size2.SetTitle(msg.Sprintf("Free: %s Trash: %s", yds.Free, yds.Trash))
			if yds.ChLast { // last synchronized list changed
				if m.lastCancel != nil {
					m.lastCancel() // stop all click handlers
				}
				for i := range m.lasts {
					m.lasts[i].Hide()
				}
				if len(yds.Last) > 0 {
					ctx, cancel := context.WithCancel(context.Background())
					m.lastCancel = cancel
					for i, p := range yds.Last {
						short, full := tools.ShortName(p, 40), filepath.Join(YD.Path, p)
						m.lasts[i].SetTitle(short)
						m.lasts[i].SetTooltip(p)
						if tools.NotExists(full) {
							m.lasts[i].Disable()
						} else {
							m.lasts[i].Enable()
							// start individual click handler for each sub menu item
							go func(ctx context.Context, mi *systray.MenuItem, path string) {
								for {
									select {
									case <-ctx.Done():
										return
									case <-mi.ClickedCh:
										tools.XdgOpen(path)
									}
								}
							}(ctx, m.lasts[i], full)
						}
						m.lasts[i].Show()
					}
					m.last.Enable()
				} else {
					m.last.Disable()
				}
				m.last.Show()
				llog.Debug("Last synchronized length:", len(yds.Last))
			}
			if yds.Stat != yds.Prev { // status changed
				// change indicator icon
				switch yds.Stat {
				case "idle":
					systray.SetIcon(icons.IdleIcon)
				case "busy", "index":
					systray.SetIcon(icons.BusyIcons[currentIcon])
					if yds.Prev != "busy" && yds.Prev != "index" {
						tick.Reset(333 * time.Millisecond)
					}
				case "none", "paused":
					systray.SetIcon(icons.PauseIcon)
				default:
					systray.SetIcon(icons.ErrorIcon)
				}
				// handle Start/Stop menu title
				if yds.Stat == "none" {
					m.start.Enable()
					m.start.Show()
					m.stop.Hide()
					m.out.Disable()
				} else if yds.Prev == "none" || yds.Prev == "unknown" {
					m.stop.Enable()
					m.stop.Show()
					m.start.Hide()
					m.out.Enable()
				}
				if note { // handle notifications
					switch {
					case yds.Stat == "none" && yds.Prev != "unknown":
						notifySend(icons.IconNotify, "Yandex.Disk",
							msg.Sprintf("Daemon stopped"))
					case yds.Prev == "none":
						notifySend(icons.IconNotify, "Yandex.Disk",
							msg.Sprintf("Daemon started"))
					case (yds.Stat == "busy" || yds.Stat == "index") &&
						(yds.Prev != "busy" && yds.Prev != "index"):
						notifySend(icons.IconNotify, "Yandex.Disk",
							msg.Sprintf("Synchronization started"))
					case (yds.Stat == "idle" || yds.Stat == "error") &&
						(yds.Prev == "busy" || yds.Prev == "index"):
						notifySend(icons.IconNotify, "Yandex.Disk",
							msg.Sprintf("Synchronization finished"))
					}
				}
			}
			llog.Debug("Change handled")
		}
	}
}

func onExit() {
	llog.Debug("All done. Bye!")
}
