// Copyleft 2017-2023 Sly_tom_cat (slytomcat@mail.ru)
// License: GPL v.3

//go:generate gotext update -out catalog.go -lang=en,ru

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"path/filepath"
	"strings"
	"time"

	"github.com/slytomcat/llog"
	"github.com/slytomcat/systray"
	"github.com/slytomcat/yd-go/icons"
	"github.com/slytomcat/yd-go/notify"
	"github.com/slytomcat/yd-go/tools"
	"github.com/slytomcat/ydisk"
	"golang.org/x/text/message"
)

var (
	version = "local build"

	msg             *message.Printer  // msg is the Localization printer
	statusTr        map[string]string // translated statuses
	icon            *icons.Icon       // icon helper
	notifySend      func(title, body string)
	notifyAvailable bool
	appConfig       *tools.Config
)

const (
	appName = "yd-go"
	about   = appName + ` is the panel indicator for Yandex.Disk daemon.

	Version: %s

Copyleft 2017-%s Sly_tom_cat (slytomcat@mail.ru)

	License: GPL v.3

`
)

type menu struct {
	status      *systray.MenuItem     // menu item to show current status
	size1       *systray.MenuItem     // menu item to show used/total sizes
	size2       *systray.MenuItem     // menu item to show free anf trash sizes
	last        *systray.MenuItem     // Sub-menu with last synchronized
	lastMItem   [10]*systray.MenuItem // last synchronized menu items
	lastPath    [10]string            // paths to last synchronized
	start       *systray.MenuItem
	stop        *systray.MenuItem
	out         *systray.MenuItem
	path        *systray.MenuItem
	notes       *systray.MenuItem
	theme       *systray.MenuItem
	daemonStart *systray.MenuItem
	daemonStop  *systray.MenuItem
	site        *systray.MenuItem
	help        *systray.MenuItem
	about       *systray.MenuItem
	donate      *systray.MenuItem
	quit        *systray.MenuItem
	warning     *systray.MenuItem
}

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	// Initialize application and get the application configuration
	cfgPath := tools.AppInit(appName, os.Args, version)
	appConfig = tools.NewConfig(cfgPath)

	// Initialize translations
	lng := os.Getenv("LANG")
	if len(lng) > 2 {
		lng = lng[:2]
	}

	llog.Debugf("Local language is: %v", lng)
	msg = message.NewPrinter(message.MatchLanguage(lng))

	// Initialize icon helper
	icon = icons.NewIcon(appConfig.Theme, systray.SetIcon)

	// Initialize notifications
	notifyHandler, err := notify.New(appName, icon.NotifyIcon, true, -1)
	if err != nil {
		notifyAvailable = false
		notifySend = func(title, body string) {}
		appConfig.Notifications = false
		llog.Warningf("Notification is not available due to D-Bus connection error: %v", err)
	} else {
		notifyAvailable = true
		notifySend = func(title, body string) {
			llog.Debug("Message:", title, ":", body)
			notifyHandler.Send("", title, body)
		}
	}

	// Initialize status localization
	statusTr = map[string]string{
		"idle":   msg.Sprintf("idle"),
		"index":  msg.Sprintf("index"),
		"busy":   msg.Sprintf("busy"),
		"none":   msg.Sprintf("none"),
		"paused": msg.Sprintf("paused"),
	}

	// Initialize systray menu
	m := new(menu)
	systray.SetTitle("Yandex.Disk")
	m.status = systray.AddMenuItem("", "")
	m.size1 = systray.AddMenuItem("", "")
	m.size2 = systray.AddMenuItem("", "")
	systray.AddSeparator()
	m.last = systray.AddMenuItem(msg.Sprintf("Last synchronized"), "")
	for i := 0; i < 10; i++ {
		m.lastMItem[i] = m.last.AddSubMenuItem("", "")
	}
	systray.AddSeparator()
	m.start = systray.AddMenuItem(msg.Sprintf("Start daemon"), "")
	m.stop = systray.AddMenuItem(msg.Sprintf("Stop daemon"), "")
	systray.AddSeparator()
	m.out = systray.AddMenuItem(msg.Sprintf("Show daemon output"), "")
	m.path = systray.AddMenuItem(msg.Sprintf("Open Yandex.Disk folder"), "")
	m.site = systray.AddMenuItem(msg.Sprintf("Open Yandex.Disk in browser"), "")
	setup := systray.AddMenuItem(msg.Sprintf("Settings"), "")
	m.theme = setup.AddSubMenuItemCheckbox(msg.Sprintf("Light theme"), "", appConfig.Theme == "light")
	m.notes = setup.AddSubMenuItemCheckbox(msg.Sprintf("Notifications"), "", appConfig.Notifications)
	m.daemonStart = setup.AddSubMenuItemCheckbox(msg.Sprintf("Start on start"), "", appConfig.StartDaemon)
	m.daemonStop = setup.AddSubMenuItemCheckbox(msg.Sprintf("Stop on exit"), "", appConfig.StopDaemon)
	systray.AddSeparator()
	m.help = systray.AddMenuItem(msg.Sprintf("Help"), "")
	m.about = systray.AddMenuItem(msg.Sprintf("About"), "")
	m.donate = systray.AddMenuItem(msg.Sprintf("Donations"), "")
	systray.AddSeparator()
	m.quit = systray.AddMenuItem(msg.Sprintf("Quit"), "")
	m.status.Disable()
	m.size1.Disable()
	m.size2.Disable()
	m.last.Disable()
	m.start.Hide()
	m.stop.Hide()
	for i := 0; i < 10; i++ {
		m.lastMItem[i].Hide()
	}
	if !notifyAvailable { // disable all menu items that are dependant on notification service
		m.about.Disable()
		m.out.Disable()
		m.notes.Disable()
		// add meny warning
		systray.AddSeparator()
		m.warning = systray.AddMenuItem(msg.Sprintf("Notification service unavailable!"), "")
	} else {
		m.warning = systray.AddMenuItem("", "")
		m.warning.Hide()
	}

	// Create new YDisk instance
	YD, err := ydisk.NewYDisk(appConfig.Conf)
	if err != nil {
		llog.Critical("Fatal error:", err)
	}

	// Start events handler
	go eventHandler(m, appConfig, YD, notifyHandler)
}

// eventHandler handles all application lifetime events
func eventHandler(m *menu, cfg *tools.Config, YD *ydisk.YDisk, notifyHandler *notify.Notify) {
	llog.Debug("event handler started")
	defer llog.Debug("event handler exited.")
	if cfg.StartDaemon {
		go YD.Start()
	}
	defer func() {
		if notifyHandler != nil {
			notifyHandler.Close()
		}
		if cfg.StopDaemon {
			YD.Stop()
		}
		YD.Close()
		systray.Quit()
	}()
	// register interrupt signals chan
	canceled := make(chan os.Signal, 1)
	signal.Notify(canceled, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case <-m.lastMItem[0].ClickedCh:
			tools.XdgOpen(m.lastPath[0])
		case <-m.lastMItem[1].ClickedCh:
			tools.XdgOpen(m.lastPath[1])
		case <-m.lastMItem[2].ClickedCh:
			tools.XdgOpen(m.lastPath[2])
		case <-m.lastMItem[3].ClickedCh:
			tools.XdgOpen(m.lastPath[3])
		case <-m.lastMItem[4].ClickedCh:
			tools.XdgOpen(m.lastPath[4])
		case <-m.lastMItem[5].ClickedCh:
			tools.XdgOpen(m.lastPath[5])
		case <-m.lastMItem[6].ClickedCh:
			tools.XdgOpen(m.lastPath[6])
		case <-m.lastMItem[7].ClickedCh:
			tools.XdgOpen(m.lastPath[7])
		case <-m.lastMItem[8].ClickedCh:
			tools.XdgOpen(m.lastPath[8])
		case <-m.lastMItem[9].ClickedCh:
			tools.XdgOpen(m.lastPath[9])
		case <-m.start.ClickedCh:
			go YD.Start()
		case <-m.stop.ClickedCh:
			go YD.Stop()
		case <-m.out.ClickedCh:
			notifySend(msg.Sprintf("Yandex.Disk daemon output"), YD.Output())
		case <-m.path.ClickedCh:
			tools.XdgOpen(YD.Path)
		case <-m.site.ClickedCh:
			tools.XdgOpen("https://disk.yandex.ru")
		case <-m.theme.ClickedCh:
			if handleCheck(m.theme) {
				cfg.Theme = "light"
			} else {
				cfg.Theme = "dark"
			}
			icon.SetTheme(cfg.Theme)
		case <-m.notes.ClickedCh:
			cfg.Notifications = handleCheck(m.notes)
		case <-m.daemonStart.ClickedCh:
			cfg.StartDaemon = handleCheck(m.daemonStart)
		case <-m.daemonStop.ClickedCh:
			cfg.StopDaemon = handleCheck(m.daemonStop)
		case <-m.help.ClickedCh:
			tools.XdgOpen("https://github.com/slytomcat/yd-go/wiki/FAQ&SUPPORT")
		case <-m.about.ClickedCh:
			notifySend("yd-go", msg.Sprintf(about, version, time.Now().Format("2006")))
		case <-m.donate.ClickedCh:
			tools.XdgOpen("https://github.com/slytomcat/yd-go/wiki/Donations")
		case <-canceled:
			llog.Warning("\nExecution is interrupted")
			return
		case <-m.quit.ClickedCh:
			llog.Debug("Exit requested")
			return
		case <-m.warning.ClickedCh:
			tools.XdgOpen("https://github.com/slytomcat/yd-go/wiki/FAQ")
		case yds := <-YD.Changes: // YDisk change event
			handleUpdate(m, &yds, YD.Path)
		}
	}
}

func handleCheck(mi *systray.MenuItem) bool {
	if mi.Checked() {
		mi.Uncheck()
		return false
	}
	mi.Check()
	return true
}

func handleUpdate(m *menu, yds *ydisk.YDvals, path string) {
	st := strings.Join([]string{statusTr[yds.Stat], yds.Prog, yds.Err, tools.MakeTitle(yds.ErrP, 30)}, " ")
	m.status.SetTitle(msg.Sprintf("Status: %s", st))
	if yds.Stat == "error" {
		m.status.SetTooltip(fmt.Sprintf("%s\nPath: %s", yds.Err, yds.ErrP))
	} else {
		m.status.SetTooltip("")
	}
	m.size1.SetTitle(msg.Sprintf("Used: %s/%s", yds.Used, yds.Total))
	m.size2.SetTitle(msg.Sprintf("Free: %s Trash: %s", yds.Free, yds.Trash))
	if yds.ChLast { // last synchronized list changed
		for i, p := range yds.Last {
			m.lastPath[i] = filepath.Join(path, p)
			m.lastMItem[i].SetTitle(tools.MakeTitle(p, 40))
			if tools.NotExists(m.lastPath[i]) {
				m.lastMItem[i].Disable()
			} else {
				m.lastMItem[i].Enable()
			}
			m.lastMItem[i].Show() // show list items
		}
		for i := len(yds.Last); i < 10; i++ {
			m.lastMItem[i].Hide() // hide the rest of list
		}
		if len(yds.Last) == 0 {
			m.last.Disable()
		} else {
			m.last.Enable()
		}
		m.last.Show() // to update parent item view
		llog.Debug("Last synchronized length:", len(yds.Last))
	}
	if yds.Stat != yds.Prev { // status changed
		// change indicator icon
		icon.Set(yds.Stat)
		// handle Start/Stop menu items
		if yds.Stat == "none" || yds.Prev == "none" || yds.Prev == "unknown" {
			if yds.Stat == "none" {
				m.start.Show()
				m.stop.Hide()
				m.out.Disable()
			} else {
				m.stop.Show()
				m.start.Hide()
				if notifyAvailable {
					m.out.Enable()
				}
			}
		}
		if appConfig.Notifications {
			go handleNotifications(yds)
		}
	}
	m.last.Show() // to update parent item view
	llog.Debug("Change handled")
}

func handleNotifications(yds *ydisk.YDvals) {
	switch {
	case yds.Stat == "none" && yds.Prev != "unknown":
		notifySend("Yandex.Disk", msg.Sprintf("Daemon stopped"))
	case yds.Prev == "none":
		notifySend("Yandex.Disk", msg.Sprintf("Daemon started"))
	case (yds.Stat == "busy" || yds.Stat == "index") &&
		(yds.Prev != "busy" && yds.Prev != "index"):
		notifySend("Yandex.Disk", msg.Sprintf("Synchronization started"))
	case (yds.Stat == "idle" || yds.Stat == "error") &&
		(yds.Prev == "busy" || yds.Prev == "index"):
		notifySend("Yandex.Disk", msg.Sprintf("Synchronization finished"))
	}
}

func onExit() {
	appConfig.Save()
	icon.CleanUp()
	llog.Debug("All done. Bye!")
}
