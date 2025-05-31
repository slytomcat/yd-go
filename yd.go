// Copyleft 2017 - +Inf Sly_tom_cat (slytomcat@mail.ru)
// License: GPL v.3

//go:generate gotext update -out catalog.go

package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"syscall"

	"path/filepath"
	"strings"
	"time"

	"github.com/slytomcat/systray"
	"github.com/slytomcat/yd-go/icons"
	"github.com/slytomcat/yd-go/notify"
	"github.com/slytomcat/yd-go/tools"
	"github.com/slytomcat/yd-go/ydisk"
	"golang.org/x/text/message"
)

var (
	version = "local build"
)

const (
	appName  = "yd-go"                 // app name for systray ID
	appTitle = "Yandex.Disk indicator" // human readable application title for icon and notifications
	about    = appName + ` is the panel indicator for Yandex.Disk daemon.

	Version: %s

Copyleft 2017-%s Sly_tom_cat (slytomcat@mail.ru)

	License: GPL v.3

`
	ydURL     = "https://disk.yandex.ru"
	faqURL    = "https://github.com/slytomcat/yd-go/wiki/FAQ"
	helpURL   = "https://github.com/slytomcat/yd-go/wiki/FAQ&SUPPORT"
	donateUrl = "https://github.com/slytomcat/yd-go/wiki/Donations"
	lastLen   = 10
)

type indicator struct {
	cfg        *tools.Config                          // app config
	msg        func(message.Reference, ...any) string // msg is the Localization printer func
	icon       *icons.Icon                            // icon helper
	notifySend func(title, msg string)                // function to send notification, nil means that notifications are not available
	log        *slog.Logger                           // logger
	menu       *menu                                  // app menu
}

type menu struct {
	status      *systray.MenuItem          // menu item to show current status
	size1       *systray.MenuItem          // menu item to show used/total sizes
	size2       *systray.MenuItem          // menu item to show free anf trash sizes
	last        *systray.MenuItem          // Sub-menu with last synchronized
	lastMItem   [lastLen]*systray.MenuItem // last synchronized menu items
	lastPath    [lastLen]string            // paths to last synchronized
	start       *systray.MenuItem          // start daemon item
	stop        *systray.MenuItem          // stop daemon item
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

func (i *indicator) makeMenu() {
	i.menu = new(menu)
	i.menu.status = systray.AddMenuItem("", "")
	i.menu.size1 = systray.AddMenuItem("", "")
	i.menu.size2 = systray.AddMenuItem("", "")
	systray.AddSeparator()
	i.menu.last = systray.AddMenuItem(i.msg("Last synchronized"), "")
	for j := range lastLen {
		l := i.menu.last.AddSubMenuItem("", "")
		l.Hide()
		i.menu.lastMItem[j] = l
	}
	systray.AddSeparator()
	i.menu.start = systray.AddMenuItem(i.msg("Start daemon"), "")
	i.menu.stop = systray.AddMenuItem(i.msg("Stop daemon"), "")
	systray.AddSeparator()
	i.menu.out = systray.AddMenuItem(i.msg("Show daemon output"), "")
	i.menu.path = systray.AddMenuItem(i.msg("Open Yandex.Disk folder"), "")
	i.menu.site = systray.AddMenuItem(i.msg("Open Yandex.Disk in browser"), "")
	setup := systray.AddMenuItem(i.msg("Settings"), "")
	i.menu.theme = setup.AddSubMenuItemCheckbox(i.msg("Light theme"), "", i.cfg.Theme == "light")
	i.menu.notes = setup.AddSubMenuItemCheckbox(i.msg("Notifications"), "", i.cfg.Notifications)
	i.menu.daemonStart = setup.AddSubMenuItemCheckbox(i.msg("Start on start"), "", i.cfg.StartDaemon)
	i.menu.daemonStop = setup.AddSubMenuItemCheckbox(i.msg("Stop on exit"), "", i.cfg.StopDaemon)
	systray.AddSeparator()
	i.menu.help = systray.AddMenuItem(i.msg("Help"), "")
	i.menu.about = systray.AddMenuItem(i.msg("About"), "")
	i.menu.donate = systray.AddMenuItem(i.msg("Donations"), "")
	systray.AddSeparator()
	i.menu.quit = systray.AddMenuItem(i.msg("Quit"), "")
	i.menu.status.Disable()
	i.menu.size1.Disable()
	i.menu.size2.Disable()
	i.menu.last.Disable()
	i.menu.start.Hide()
	i.menu.stop.Hide()
	if i.notifySend == nil { // disable all menu items that are dependant on notification service
		i.menu.about.Disable()
		i.menu.out.Disable()
		i.menu.notes.Disable()
		// add menu warning
		systray.AddSeparator()
		i.menu.warning = systray.AddMenuItem(i.msg("Notification service unavailable!"), "")
	} else {
		i.menu.warning = systray.AddMenuItem("", "")
		i.menu.warning.Hide()
	}
}

// SetupLocalization initializes translations
func SetupLocalization(logger *slog.Logger) *message.Printer {
	lng := os.Getenv("LANG")
	if len(lng) > 2 {
		lng = lng[:2]
	}
	logger.Debug("language", "LANG", lng)
	return message.NewPrinter(message.MatchLanguage(lng))
}

func main() {
	cfgPath, debug := tools.GetParams(appName, os.Args, version)
	_, id := path.Split(cfgPath)
	systray.SetID(fmt.Sprintf("%s_%s", appName, id))
	systray.Run(func() {
		defer systray.Quit() // it releases systray.Run in main()
		log := tools.SetupLogger(debug)
		cfg, err := tools.NewConfig(cfgPath)
		if err != nil {
			log.Error("config_error", "error", err)
			os.Exit(1)
		}
		defer cfg.Save()
		i := &indicator{
			cfg: cfg,
			msg: SetupLocalization(log).Sprintf,
			log: log,
		}
		// create new YDisk instance
		YD, err := ydisk.NewYDisk(i.cfg.Conf, i.log)
		if err != nil {
			i.log.Error("daemon_initialization", "error", err)
			os.Exit(1)
		}
		defer YD.Close()
		// handle starting/stopping daemon
		if i.cfg.StartDaemon {
			go YD.Start()
		}
		defer func() {
			if i.cfg.StopDaemon {
				YD.Stop()
			}
		}()
		// register interrupt signals chan
		canceled := make(chan os.Signal, 1)
		signal.Notify(canceled, syscall.SIGINT, syscall.SIGTERM)
		// set systray title
		systray.SetTitle(i.msg(appTitle))
		// initialize icon helper
		i.icon = icons.NewIcon(cfg.Theme, systray.SetIcon)
		defer i.icon.Close()
		// Initialize notifications
		if notifyHandler, err := notify.New(appName, i.icon.LogoIcon, false, -1); err != nil {
			i.notifySend = nil
			cfg.Notifications = false // disable notifications into configuration
			i.log.Warn("notifications", "status", "not_available", "error", err)
		} else {
			i.notifySend = func(title, msg string) {
				i.log.Debug("sending_message", "title", title, "message", msg)
				notifyHandler.Send(title, msg)
			}
			defer notifyHandler.Close()
		}
		// Initialize systray menu
		i.makeMenu()
		// Start events handler
		i.log.Debug("ui_event_handler", "status", "started")
		defer i.log.Debug("ui_event_handler", "status", "exited")
		for {
			select {
			case <-i.menu.lastMItem[0].ClickedCh:
				i.openPath(i.menu.lastPath[0])
			case <-i.menu.lastMItem[1].ClickedCh:
				i.openPath(i.menu.lastPath[1])
			case <-i.menu.lastMItem[2].ClickedCh:
				i.openPath(i.menu.lastPath[2])
			case <-i.menu.lastMItem[3].ClickedCh:
				i.openPath(i.menu.lastPath[3])
			case <-i.menu.lastMItem[4].ClickedCh:
				i.openPath(i.menu.lastPath[4])
			case <-i.menu.lastMItem[5].ClickedCh:
				i.openPath(i.menu.lastPath[5])
			case <-i.menu.lastMItem[6].ClickedCh:
				i.openPath(i.menu.lastPath[6])
			case <-i.menu.lastMItem[7].ClickedCh:
				i.openPath(i.menu.lastPath[7])
			case <-i.menu.lastMItem[8].ClickedCh:
				i.openPath(i.menu.lastPath[8])
			case <-i.menu.lastMItem[9].ClickedCh:
				i.openPath(i.menu.lastPath[9])
			case <-i.menu.start.ClickedCh:
				go YD.Start()
			case <-i.menu.stop.ClickedCh:
				go YD.Stop()
			case <-i.menu.out.ClickedCh:
				i.notifySend(i.msg("Yandex.Disk daemon output"), YD.Output())
			case <-i.menu.path.ClickedCh:
				i.openPath(YD.Path)
			case <-i.menu.site.ClickedCh:
				i.openPath(ydURL)
			case <-i.menu.theme.ClickedCh:
				i.cfg.Theme = i.handleThemeClick(i.menu.theme)
			case <-i.menu.notes.ClickedCh:
				i.cfg.Notifications = handleCheck(i.menu.notes)
			case <-i.menu.daemonStart.ClickedCh:
				i.cfg.StartDaemon = handleCheck(i.menu.daemonStart)
			case <-i.menu.daemonStop.ClickedCh:
				i.cfg.StopDaemon = handleCheck(i.menu.daemonStop)
			case <-i.menu.help.ClickedCh:
				i.openPath(helpURL)
			case <-i.menu.about.ClickedCh:
				i.notifySend(i.msg(appTitle), i.msg(about, version, time.Now().Format("2006")))
			case <-i.menu.donate.ClickedCh:
				i.openPath(donateUrl)
			case <-i.menu.warning.ClickedCh:
				i.openPath(faqURL)
			case yds := <-YD.Changes: // YDisk change event
				i.handleUpdate(&yds, YD.Path)
			case sig := <-canceled: // SIGINT or SIGTERM signal received
				fmt.Println() // to leave ^C on previous line
				i.log.Warn("exit", "signal", sig)
				return
			case <-i.menu.quit.ClickedCh:
				i.log.Debug("exit", "status", "requested")
				return
			}
		}
	}, nil)
}

func (i *indicator) openPath(path string) {
	if err := tools.XdgOpen(path); err != nil {
		i.log.Error("opening", "path", path, "error", err)
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

func (i *indicator) handleThemeClick(mi *systray.MenuItem) (theme string) {
	if handleCheck(mi) {
		theme = "light"
	} else {
		theme = "dark"
	}
	i.icon.SetTheme(theme)
	return
}

func joinNonEmpty(items ...string) string {
	s := strings.Builder{}
	for _, i := range items {
		if len(i) > 0 {
			s.WriteString(i)
			s.WriteString(" ")
		}
	}
	if s.Len() > 0 {
		return s.String()[:s.Len()-1]
	}
	return ""
}

// handleUpdate changes icon/menu and sends notifications if they are enabled
func (i *indicator) handleUpdate(yds *ydisk.YDvals, path string) {
	st := joinNonEmpty(i.msg(yds.Stat), yds.Prog, yds.Err, tools.MakeTitle(yds.ErrP, 30))
	i.menu.status.SetTitle(i.msg("Status: %s", st))
	i.menu.size1.SetTitle(i.msg("Used: %s/%s", yds.Used, yds.Total))
	i.menu.size2.SetTitle(i.msg("Free: %s Trash: %s", yds.Free, yds.Trash))
	if yds.ChLast { // last synchronized list changed
		for l := range lastLen {
			if l < len(yds.Last) {
				p := yds.Last[l]
				i.menu.lastPath[l] = filepath.Join(path, p)
				i.menu.lastMItem[l].SetTitle(tools.MakeTitle(p, 40))
				if tools.NotExists(i.menu.lastPath[l]) {
					i.menu.lastMItem[l].Disable()
				} else {
					i.menu.lastMItem[l].Enable()
				}
				i.menu.lastMItem[l].Show() // show list items
			} else {
				i.menu.lastMItem[l].Hide() // hide the rest of list
			}
		}
		if len(yds.Last) == 0 {
			i.menu.last.Disable()
		} else {
			i.menu.last.Enable()
		}
		i.menu.last.Show() // to update parent item view
	}
	yds.Stat = index2Busy(yds.Stat) // index and busy statuses are equal in terms of icons and notifications
	yds.Prev = index2Busy(yds.Prev)
	if yds.Stat != yds.Prev { // status changed
		// change indicator icon
		i.icon.Set(none2Paused(yds.Stat)) // index were converted to busy earlier
		// handle Start/Stop menu items
		if yds.Stat == "none" || yds.Prev == "none" || yds.Prev == "unknown" {
			if yds.Stat == "none" {
				i.menu.start.Show()
				i.menu.stop.Hide()
				i.menu.out.Disable()
			} else {
				i.menu.stop.Show()
				i.menu.start.Hide()
				if i.notifySend != nil {
					i.menu.out.Enable()
				}
			}
		}
		if i.cfg.Notifications && i.notifySend != nil {
			go i.handleNotifications(yds)
		}
	}
	i.log.Debug("ui_change", "status", "handled", "last", len(yds.Last))
}

// index2Busy converts index to busy
func index2Busy(status string) string {
	if status == "index" {
		return "busy"
	}
	return status
}

// none2Paused converts none to paused
func none2Paused(status string) string {
	if status == "none" {
		return "paused"
	}
	return status
}

func (i *indicator) handleNotifications(yds *ydisk.YDvals) {
	switch {
	case yds.Stat == "none" && yds.Prev != "unknown":
		i.notifySend(i.msg(appTitle), i.msg("Daemon stopped"))
	case yds.Prev == "none":
		i.notifySend(i.msg(appTitle), i.msg("Daemon started"))
	case yds.Prev != "busy" && yds.Stat == "busy":
		i.notifySend(i.msg(appTitle), i.msg("Synchronization started"))
	case yds.Prev == "busy" && yds.Stat != "busy":
		i.notifySend(i.msg(appTitle), i.msg("Synchronization finished"))
	}
}
