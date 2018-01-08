package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/slytomcat/YD.go/YDisk"
	"github.com/slytomcat/YD.go/icons"
	"github.com/slytomcat/confJSON"
	"github.com/slytomcat/systray"
)

func notExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return os.IsNotExist(err)
	}
	return false
}

func expandHome(path string) string {
	if len(path) == 0 || path[0] != '~' {
		return path
	}
	usr, err := user.Current()
	if err != nil {
		log.Fatal("Can't get current user profile:", err)
	}
	return filepath.Join(usr.HomeDir, path[1:])
}

func xdgOpen(uri string) {
	err := exec.Command("xdg-open", uri).Start()
	if err != nil {
		log.Println(err)
	}
}

func notifySend(icon, title, body string) {
	err := exec.Command("notify-send", "-i", icon, title, body).Start()
	if err != nil {
		log.Println(err)
	}
}

func shortName(f string, l int) string {
	v := []rune(f)
	if len(v) > l {
		n := (l - 3) / 2
		k := n
		if n+k+3 < l {
			k += 1
		}
		return string(v[:n]) + "..." + string(v[len(v)-k:])
	} else {
		return f
	}
}

func checkDaemon(conf string) string {
	// Check that yandex-disk daemon is installed (exit if not)
	if notExists("/usr/bin/yandex-disk") {
		log.Fatal("Yandex.Disk CLI utility is not installed. Install it first.")
	}
	f, err := os.Open(conf)
	if err != nil {
		log.Fatal("Daemon configuration file opening error:", err)
	}
	defer f.Close()
	reader := io.Reader(f)
	line := ""
	dir := ""
	auth := ""
	for {
		n, _ := fmt.Fscanln(reader, &line)
		if n == 0 {
			break
		}
		if strings.HasPrefix(line, "dir") {
			dir = line[5 : len(line)-1]
		}
		if strings.HasPrefix(line, "auth") {
			auth = line[6 : len(line)-1]
		}
		if dir != "" && auth != "" {
			break
		}
	}
	if notExists(dir) || notExists(auth) {
		log.Fatal("Daemon is not configured. Run:\nyandex-disk setup")
	}
	return dir
}

func onReady() {
	// Prepare the application configuration
	// Make default app configuration values
	AppCfg := map[string]interface{}{
		"Conf":          expandHome("~/.config/yandex-disk/config.cfg"), // path to daemon config file
		"Theme":         "dark",                                         // icons theme name
		"Notifications": true,                                           // display desktop notification
		"StartDaemon":   true,                                           // start daemon on app start
		"StopDaemon":    false,                                          // stop daemon on app closure
	}
	// Check that app configuration file path exists
	AppConfigHome := expandHome("~/.config/yd.go")
	if notExists(AppConfigHome) {
		err := os.MkdirAll(AppConfigHome, 0766)
		if err != nil {
			log.Fatal("Can't create application configuration path:", err)
		}
	}
	// Check that app configuration file exists
	AppConfigFile := filepath.Join(AppConfigHome, "default.cfg")
	if notExists(AppConfigFile) {
		//Create and save new configuration file with default values
		confJSON.Save(AppConfigFile, AppCfg)
	} else {
		// Read app configuration file
		confJSON.Load(AppConfigFile, &AppCfg)
	}
	// Check that daemon installed and configured
	FolderPath := checkDaemon(AppCfg["Conf"].(string))
	// Initialize icon theme
	icons.SetTheme("/usr/share/yd.go", AppCfg["Theme"].(string))
	// Initialize systray icon
	systray.SetIcon(icons.IconPause)
	systray.SetTitle("")
	// Initialize systray menu
	mStatus := systray.AddMenuItem("Status: unknown", "")
	mStatus.Disable()
	mSize1 := systray.AddMenuItem("Used: .../...", "")
	mSize1.Disable()
	mSize2 := systray.AddMenuItem("Free: ... Trash: ...", "")
	mSize2.Disable()
	systray.AddSeparator()
	// use ZERO WIDTH SPACE to avoid maching with filename
	mLast := systray.AddMenuItem("Last synchronized"+"\u200B", "")
	mLast.Disable()
	systray.AddSeparator()
	mStartStop := systray.AddMenuItem("", "") // no title at start as current status is unknown
	systray.AddSeparator()
	mPath := systray.AddMenuItem("Open path: "+FolderPath, "")
	mSite := systray.AddMenuItem("Open YandexDisk in browser", "")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "")
	/*TO_DO:
	 * Additional menu items:
	 * 1. About ???
	 * 2. Help -> redirect to github wiki page "FAQ and how to report issue"
	 * 3. Show daemon output -> window/notify??
	 * */
	//  Create new YDisk interface
	YD := YDisk.NewYDisk(AppCfg["Conf"].(string), FolderPath)
	var Last map[string]string
	go func() {
		log.Println("Menu handler started")
		defer log.Println("Menu handler exited.")
		// defer request for exit from systray main loop (gtk.main())
		defer systray.Quit()
		for {
			select {
			case title := <-mStartStop.ClickedCh:
				switch title {
				case "Start":
					YD.Start()
				case "Stop":
					YD.Stop()
				} // do nothing in other cases
			case title := <-mLast.ClickedCh:
				if title != "Last synchronized"+"\u200B" {
					xdgOpen(filepath.Join(FolderPath, Last[title]))
				}
			case <-mPath.ClickedCh:
				xdgOpen(FolderPath)
			case <-mSite.ClickedCh:
				xdgOpen("https://disk.yandex.com")
			case <-mQuit.ClickedCh:
				log.Println("Exit requested.")
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
		log.Println("Changes handler started")
		defer log.Println("Changes handler exited.")
		// Prepare the staff for icon animation
		currentIcon := 0
		tick := time.NewTimer(333 * time.Millisecond)
		defer tick.Stop()
		// Start daemon if it is configured
		if AppCfg["StartDaemon"].(bool) {
			YD.Start()
		}
		currentStatus := ""
		for {
			select {
			case yds, ok := <-YD.Changes: // YD changed status event
				if !ok { // as Changes channel closed - exit
					return
				}
				currentStatus = yds.Stat

				mStatus.SetTitle("Status: " + yds.Stat + " " + yds.Prog +
					yds.Err + " " + shortName(yds.ErrP, 30))
				mSize1.SetTitle("Used: " + yds.Used + "/" + yds.Total)
				mSize2.SetTitle("Free: " + yds.Free + " Trash: " + yds.Trash)
				// handle last synchronized
				if yds.ChLast {
					mLast.RemoveSubmenu()
					Last = make(map[string]string, 10)
					last := []systray.SubmenuItem{}
					for _, p := range yds.Last {
						short := shortName(p, 40)
						Last[short] = p
						last = append(last, systray.SubmenuItem{short, !notExists(p)})
					}
					if len(last) > 0 {
						mLast.AddSubmenu(last)
						mLast.Enable()
					} else {
						mLast.Disable()
					}
				}
				switch yds.Stat {
				case "idle":
					systray.SetIcon(icons.IconIdle)
				case "busy", "index":
					systray.SetIcon(icons.IconBusy[currentIcon])
					tick.Reset(333 * time.Millisecond)
				case "none", "paused":
					systray.SetIcon(icons.IconPause)
				default:
					systray.SetIcon(icons.IconError)
				}
				if yds.Stat != yds.Prev { // status changed
					// handle Start/Stop menu title
					if yds.Stat == "none" {
						mStartStop.SetTitle("Start")
					} else if mStartStop.GetTitle() != "Stop" {
						mStartStop.SetTitle("Stop")
					}
					// Handle notifications
					if AppCfg["Notifications"].(bool) {
						switch {
						case yds.Stat == "none" && yds.Prev != "unknown":
							notifySend(icons.IconNotify, "Yandex.Disk", "Daemon stopped")
						case yds.Prev == "none":
							notifySend(icons.IconNotify, "Yandex.Disk", "Daemon started")
						case (yds.Stat == "busy" || yds.Stat == "index") &&
							(yds.Prev != "busy" && yds.Prev != "index"):
							notifySend(icons.IconNotify, "Yandex.Disk", "Synchronization started")
						case (yds.Stat == "idle" || yds.Stat == "error") &&
							(yds.Prev == "busy" || yds.Prev == "index"):
							notifySend(icons.IconNotify, "Yandex.Disk", "Synchronization finished")
						}
					}
				}
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

func onExit() {}

func main() {
	/* Initialize logging facility */
	log.SetOutput(os.Stderr)
	log.SetPrefix("")
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)

	systray.Run(onReady, onExit)
}
