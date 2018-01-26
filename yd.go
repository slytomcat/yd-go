package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/slytomcat/confJSON"
	"github.com/slytomcat/llog"
	"github.com/slytomcat/systray"
	"github.com/slytomcat/yd-go/icons"
	"github.com/slytomcat/yd-go/tools"
	"github.com/slytomcat/yd-go/ydisk"
	"golang.org/x/text/message"
)

const about = `yd-go is the panel indicator for Yandex.Disk daemon.

      Version: 0.3

Copyleft 2017-2018 Sly_tom_cat (slytomcat@mail.ru)

	  License: GPL v.3

`

var (
	// AppConfigFile stores the application configuration file path
	AppConfigFile string
	// Msg is the Localozation printer
	Msg *message.Printer
)

func notifySend(icon, title, body string) {
	llog.Debug("Message:", title, ":", body)
	err := exec.Command("notify-send", "-i", icon, title, body).Start()
	if err != nil {
		llog.Error(err)
	}
}

// LastT type is just map[strig]string protected by RWMutex to be read and updated
// form different goroutines simulationusly
type LastT struct {
	m map[string]*string
	l sync.RWMutex
}

func (l *LastT) reset() {
	l.l.Lock()
	l.m = make(map[string]*string, 10) // 10 - is a maximum lenghth of the last synchronized
	l.l.Unlock()
}

func (l *LastT) update(key, value string) {
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

func init() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Allow debugging messages to be sent to stderr")
	flag.StringVar(&AppConfigFile, "config", "~/.config/yd-go/default.cfg", "Path to the indicator configuration file")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n\n\t\tyd-go [-debug] [-config=<Path to indicator config>]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	// Initialize logging facility
	llog.SetOutput(os.Stderr)
	llog.SetPrefix("")
	llog.SetFlags(log.Lshortfile | log.Lmicroseconds)
	if debug {
		llog.SetLevel(llog.DEBUG)
		llog.Info("Debugging enabled")
	} else {
		llog.SetLevel(-1)
	}
	// Initialize translations
	Msg = message.NewPrinter(message.MatchLanguage("ru"))
}

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	// Prepare the application configuration
	// Make default app configuration values
	AppCfg := map[string]interface{}{
		"Conf":          tools.ExpandHome("~/.config/yandex-disk/config.cfg"), // path to daemon config file
		"Theme":         "dark",                                               // icons theme name
		"Notifications": true,                                                 // display desktop notification
		"StartDaemon":   true,                                                 // start daemon on app start
		"StopDaemon":    false,                                                // stop daemon on app closure
	}
	// Check that app configuration file path exists
	AppConfigHome := tools.ExpandHome("~/.config/yd-go")
	if tools.NotExists(AppConfigHome) {
		err := os.MkdirAll(AppConfigHome, 0766)
		if err != nil {
			llog.Critical("Can't create application configuration path:", err)
		}
	}
	// Path to app configuration file path always comes from command-line flag
	AppConfigFile = tools.ExpandHome(AppConfigFile)
	llog.Debug("Configuration:", AppConfigFile)
	// Check that app configuration file exists
	if tools.NotExists(AppConfigFile) {
		//Create and save new configuration file with default values
		confJSON.Save(AppConfigFile, AppCfg)
	} else {
		// Read app configuration file
		confJSON.Load(AppConfigFile, &AppCfg)
	}
	// Create new ydisk interface
	YD := ydisk.NewYDisk(AppCfg["Conf"].(string))
	// Start daemon if it is configured
	if AppCfg["StartDaemon"].(bool) {
		go YD.Start()
	}
	// Initialize icon theme
	icons.SetTheme("/usr/share/yd-go/icons", AppCfg["Theme"].(string))
	// Initialize systray icon
	systray.SetIcon(icons.IconPause)
	systray.SetTitle("")
	// Initialize systray menu
	mStatus := systray.AddMenuItem(Msg.Sprint("Status: ")+Msg.Sprint("unknown"), "")
	mStatus.Disable()
	mSize1 := systray.AddMenuItem("", "")
	mSize1.Disable()
	mSize2 := systray.AddMenuItem("", "")
	mSize2.Disable()
	systray.AddSeparator()
	// use 2 ZERO WIDTH SPACES to avoid matching with filenames
	mLast := systray.AddMenuItem("\u200B\u2060"+Msg.Sprint("Last synchronized"), "")
	mLast.Disable()
	systray.AddSeparator()
	mStartStop := systray.AddMenuItem("", "") // no title at start as current status is unknown
	systray.AddSeparator()
	mOutput := systray.AddMenuItem(Msg.Sprint("Show daemon output"), "")
	mPath := systray.AddMenuItem(Msg.Sprint("Open: ")+YD.Path, "")
	mSite := systray.AddMenuItem(Msg.Sprint("Open YandexDisk in browser"), "")
	systray.AddSeparator()
	mHelp := systray.AddMenuItem(Msg.Sprint("Help"), "")
	mAbout := systray.AddMenuItem(Msg.Sprint("About"), "")
	mDon := systray.AddMenuItem(Msg.Sprint("Donations"), "")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem(Msg.Sprint("Quit"), "")
	// Dictionary for last synchronized title (as shorten path) and full path
	var last LastT

	go func() {
		llog.Debug("Menu handler started")
		defer llog.Debug("Menu handler exited.")
		for {
			select {
			case title := <-mStartStop.ClickedCh:
				switch {
				case strings.HasPrefix(title, "\u200B"): // start
					go YD.Start()
				case strings.HasPrefix(title, "\u2060"): // stop
					go YD.Stop()
				} // do nothing in other cases
			case title := <-mLast.ClickedCh:
				if !strings.HasPrefix(title, "\u200B\u2060") {
					tools.XdgOpen(last.get(title))
				}
			case <-mOutput.ClickedCh:
				notifySend(icons.IconNotify, Msg.Sprint("Yandex.Disk daemon output"), YD.Output())
			case <-mPath.ClickedCh:
				tools.XdgOpen(YD.Path)
			case <-mSite.ClickedCh:
				tools.XdgOpen(Msg.Sprint("https://disk.yandex.com"))
			case <-mHelp.ClickedCh:
				tools.XdgOpen(Msg.Sprint("https://github.com/slytomcat/YD.go/wiki/FAQ&SUPPORT"))
			case <-mAbout.ClickedCh:
				notifySend(icons.IconNotify, " ", about)
			case <-mDon.ClickedCh:
				tools.XdgOpen(Msg.Sprint("https://github.com/slytomcat/yd-go/wiki/Donats"))
			case <-mQuit.ClickedCh:
				llog.Debug("Exit requested.")
				// Stop daemon if it is configured
				if AppCfg["StopDaemon"].(bool) {
					YD.Stop()
				}
				YD.Close() // it closes Changes channel
				return
			}
		}
	}()

	go func() {
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
			case yds, ok := <-YD.Changes: // YD changed status event
				if !ok { // as Changes channel closed - exit
					return
				}
				currentStatus = yds.Stat

				mStatus.SetTitle(Msg.Sprint("Status: ") + Msg.Sprint(yds.Stat) + " " + yds.Prog +
					yds.Err + " " + tools.ShortName(yds.ErrP, 30))
				mSize1.SetTitle(Msg.Sprintf("Used: %s/%s", yds.Used, yds.Total))
				mSize2.SetTitle(Msg.Sprintf("Free: %s Trash: %s", yds.Free, yds.Trash))
				if yds.ChLast { // last synchronized list changed
					mLast.RemoveSubmenu()
					last.reset()
					if len(yds.Last) > 0 {
						for _, p := range yds.Last {
							short, full := tools.ShortName(p, 40), filepath.Join(YD.Path, p)
							mLast.AddSubmenuItem(short, tools.NotExists(full))
							last.update(short, full)
						}
						mLast.Enable()
					} else {
						mLast.Disable()
					}
					llog.Debug("Last synchronized updated L", last.len())
				}
				if yds.Stat != yds.Prev { // status changed
					// change indicator icon
					switch yds.Stat {
					case "idle":
						systray.SetIcon(icons.IconIdle)
					case "busy", "index":
						systray.SetIcon(icons.IconBusy[currentIcon])
						if yds.Prev != "busy" && yds.Prev != "index" {
							tick.Reset(333 * time.Millisecond)
						}
					case "none", "paused":
						systray.SetIcon(icons.IconPause)
					default:
						systray.SetIcon(icons.IconError)
					}
					// handle Start/Stop menu title
					if yds.Stat == "none" {
						mStartStop.SetTitle("\u200B" + Msg.Sprint("Start daemon"))
						mOutput.Disable()
					} else if yds.Prev == "none" || yds.Prev == "unknown" {
						mStartStop.SetTitle("\u2060" + Msg.Sprint("Stop daemon"))
						mOutput.Enable()
					}
					// handle notifications
					if AppCfg["Notifications"].(bool) {
						switch {
						case yds.Stat == "none" && yds.Prev != "unknown":
							notifySend(
								icons.IconNotify,
								Msg.Sprint("Yandex.Disk"),
								Msg.Sprint("Daemon stopped"))
						case yds.Prev == "none":
							notifySend(
								icons.IconNotify,
								Msg.Sprint("Yandex.Disk"),
								Msg.Sprint("Daemon started"))
						case (yds.Stat == "busy" || yds.Stat == "index") &&
							(yds.Prev != "busy" && yds.Prev != "index"):
							notifySend(
								icons.IconNotify,
								Msg.Sprint("Yandex.Disk"),
								Msg.Sprint("Synchronization started"))
						case (yds.Stat == "idle" || yds.Stat == "error") &&
							(yds.Prev == "busy" || yds.Prev == "index"):
							notifySend(
								icons.IconNotify,
								Msg.Sprint("Yandex.Disk"),
								Msg.Sprint("Synchronization finished"))
						}
					}
				}
				llog.Debug("Change handled")
			case <-tick.C: //  timer event
				currentIcon++
				currentIcon %= 5
				if currentStatus == "busy" || currentStatus == "index" {
					systray.SetIcon(icons.IconBusy[currentIcon])
					tick.Reset(333 * time.Millisecond)
				}
			}
		}
	}()
}

func onExit() {
	llog.Debug("All done. Bye!")
}
