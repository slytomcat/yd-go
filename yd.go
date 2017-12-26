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

	. "github.com/slytomcat/YD.go/YDisk"
	. "github.com/slytomcat/YD.go/icons"
	. "github.com/slytomcat/confJSON"
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
	err := exec.Command("xdg-open", uri).Run()
	if err != nil {
		log.Println(err)
	}
}

func notifySend(icon, title, body string) {
	exec.Command("notify-send", "-i", icon, title, body).Run()
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
	for n, _ := fmt.Fscanln(reader, &line); n > 0; {
		//fmt.Println(line)
		if strings.HasPrefix(line, "dir") {
			dir = line[5 : len(line)-1]
		}
		if strings.HasPrefix(line, "auth") {
			auth = line[6 : len(line)-1]
		}
		if dir != "" && auth != "" {
			break
		}
		n, _ = fmt.Fscanln(reader, &line)
	}
	if notExists(dir) || notExists(auth) {
		log.Fatal("Daemon is not configured.")
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
	// Check that app config file path exists
	AppConfigHome := expandHome("~/.config/yd.go")
	if notExists(AppConfigHome) {
		err := os.MkdirAll(AppConfigHome, 0766)
		if err != nil {
			log.Fatal("Can't create application config path:", err)
		}
	}
	// Check tha app config file exists
	AppConfigFile := filepath.Join(AppConfigHome, "default.cfg")
	if notExists(AppConfigFile) {
		//Create and save new config file with default values
		Save(AppConfigFile, AppCfg)
	} else {
		// Read app config file
		Load(AppConfigFile, &AppCfg)
	}
	FolderPath := checkDaemon(AppCfg["Conf"].(string))
	// Initialize icon theme
	SetTheme("/usr/share/yd.go", AppCfg["Theme"].(string))
	// Initialize systray icon
	systray.SetIcon(IconPause)
	systray.SetTitle("")
	// Initialize systray menu
	mStatus := systray.AddMenuItem("Status: unknown", "")
	mStatus.Disable()
	mSize1 := systray.AddMenuItem("Used: .../...", "")
	mSize1.Disable()
	mSize2 := systray.AddMenuItem("Free: ... Trash: ...", "")
	mSize2.Disable()
	systray.AddSeparator()
	mPath := systray.AddMenuItem("Open path: "+FolderPath, "")
	mSite := systray.AddMenuItem("Open YandexDisk in browser", "")
	systray.AddSeparator()
	mStart := systray.AddMenuItem("Start", "")
	mStart.Disable()
	mStop := systray.AddMenuItem("Stop", "")
	mStop.Disable()
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "")
	/*TO_DO:
	 * Additional menu items:
	 * 1. About ???
	 * 2. Help -> redirect to github wiki page "FAQ and how to report issue"
	 * 3. LastSynchronized submenu ??? need support from systray.C module side
	 * */
	//  create new YDisk interface
	YD := NewYDisk(AppCfg["Conf"].(string), FolderPath)
	// make go-routine for menu treatment
	go func() {
		log.Println("Munu handler started")
		defer log.Println("Munu handler started")
		if AppCfg["StartDaemon"].(bool) {
			YD.Start()
		}
		for {
			select {
			case <-mStart.ClickedCh:
				YD.Start()
			case <-mStop.ClickedCh:
				YD.Stop()
			case <-mPath.ClickedCh:
				xdgOpen(FolderPath)
			case <-mSite.ClickedCh:
				xdgOpen("https://disk.yandex.com")
			case <-mQuit.ClickedCh:
				log.Println("Exit requested.")
				if AppCfg["StopDaemon"].(bool) {
					YD.Stop()
				}
				YD.Close()
				systray.Quit()
				return
			}
		}
	}()

	//  start go-routine to display status changes in icon/menu
	go func() {
		log.Println("Status updater started")
		defer log.Println("Status updater exited.")
		currentStatus := ""
		currentIcon := 0
		tick := time.NewTimer(333 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case yds, ok := <-YD.Updates:
				if !ok {
					return
				} else {
					currentStatus = yds.Stat
					mStatus.SetTitle("Status: " + yds.Stat + " " + yds.Prog)
					mSize1.SetTitle("Used: " + yds.Used + "/" + yds.Total)
					mSize2.SetTitle("Free: " + yds.Free + " Trash: " + yds.Trash)
					switch yds.Stat {
					case "idle":
						systray.SetIcon(IconIdle)
					case "none":
						systray.SetIcon(IconPause)
						mStop.Disable()
						mStart.Enable()
					case "paused":
						systray.SetIcon(IconPause)
					case "busy":
						systray.SetIcon(IconBusy[currentIcon])
						tick.Reset(333 * time.Millisecond)
					case "index":
						systray.SetIcon(IconBusy[currentIcon])
						tick.Reset(333 * time.Millisecond)
					default:
						systray.SetIcon(IconError)
					}
					if yds.Stat != "none" {
						mStart.Disable()
						mStop.Enable()
					}
					// Handle notifications
					if AppCfg["Notifications"].(bool) && yds.Stat != yds.Prev {
						if (yds.Stat == "busy" || yds.Stat == "index") &&
							(yds.Prev != "busy" && yds.Prev != "index") {
							notifySend(IconNotify, "Yandex.Disk", "Syncronization started")
						}
						if (yds.Stat == "idle" || yds.Stat == "error") &&
							(yds.Prev == "busy" || yds.Prev == "index") {
							notifySend(IconNotify, "Yandex.Disk", "Syncronization finished")
						}
						if yds.Stat == "none" && yds.Prev != "unknown" {
							notifySend(IconNotify, "Yandex.Disk", "Daemon stopped")
						}
						if yds.Prev == "none" {
							notifySend(IconNotify, "Yandex.Disk", "Daemon started")
						}
					}
				}
			case <-tick.C:
				currentIcon++
				currentIcon %= 5
				if currentStatus == "busy" || currentStatus == "index" {
					systray.SetIcon(IconBusy[currentIcon])
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
