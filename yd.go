package main

import (
  //"log"
  "os"
  "io"
  "path"
  "time"
  "strings"
  "fmt"

  . "github.com/slytomcat/YD.go/YDisk"
  . "github.com/slytomcat/YD.go/icons"
  "github.com/slytomcat/systray"

)

/* Initialize default logger */
//var Logger *log.Logger = log.New(os.Stderr, "", log.Lshortfile|log.Lmicroseconds) // | log.Lmicroseconds)

func notExists(path string) bool {
  _, err := os.Stat(path)
  if err != nil {
    return os.IsNotExist(err)
  }
  return false
}

func checkDaemon(conf string) string {
  // Check that yandex-disk daemon is installed (exit if not)
  if notExists("/usr/bin/yandex-disk") {
    Logger.Fatal("Yandex.Disk CLI utility is not installed. Install it first.")
  }
  f, err := os.Open(conf)
  if err != nil {
    Logger.Fatal("Daemon configuration file opening error:", err)
  }
  defer f.Close()
  reader := io.Reader(f)
  line := ""
  dir := ""
  auth := ""
  for n, _ := fmt.Fscanln(reader, &line); n>0; {
    //fmt.Println(line)
    if strings.HasPrefix(line, "dir") {
      dir = line[5:len(line)-1]
    }
    if strings.HasPrefix(line, "auth") {
      auth = line[6:len(line)-1]
    }
    if dir != "" && auth != "" {
      break
    }
    n, _ = fmt.Fscanln(reader, &line)
  }
  if notExists(dir) || notExists(auth) {
    Logger.Fatal("Daemon is not configured.")
  }
  return dir
}

func main() {
  systray.Run(onReady, onExit)
}

func onReady() {

  AppConfigHome := path.Join(os.Getenv("HOME"),".config/yd.go")
  if notExists(AppConfigHome) {
    err := os.MkdirAll(AppConfigHome, 0766)
    if err != nil {
      Logger.Fatal("Can't create aplication config path.")
    }
  }
  AppConfigFile := path.Join(AppConfigHome, "default.cfg")
  if notExists(AppConfigFile) {
    //TO_DO: create and save new config file with default values)
  }
  // TO_DO: read app config file
  Conf := "/home/stc/.config/yandex-disk/config.cfg"
  Theme := "dark"
  StartDaemon := true
  StopDaemon := false

  Path := checkDaemon(Conf)
  /* TO_DO:
   * 1. Read/create_and_save app configuration file (~/.config/yd.go/default.cfg) for:
   *  path to daemon config (default config="/home/stc/.config/yandex-disk/config.cfg")
   *  icons theme (default theme="dark")
   *  autostart indicator (default autostart="yes")
   *  start daemon on start (default startdaemon="yes")
   *  stop daemon on exit (default stopdaemon="no")
   * 2. Read daemon config for:
   *  path to synchronized folder
   *  path to auth file
   * 3. Check that daemon is configured (check auth and conf paths existance, exit if not)
   * */
  // Initialize icon theme
  SetTheme(Theme)    // theme should be read from app config
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
   * 4. Open local folder
   * 5. Open yandex.disk in browser
   * */
  //  create new YDisk interface
  YD := NewYDisk(Conf, Path)
  // make go-routine for menu treatment
  go func(){
    if StartDaemon {
      YD.Start()
    }
    for {
      select {
      case <-mStart.ClickedCh:
        YD.Start()
      case <-mStop.ClickedCh:
        YD.Stop()
      case <-mQuit.ClickedCh:
        Logger.Println("Exit requested.")
        if StopDaemon {
          YD.Stop()
        }
        YD.Close()
        systray.Quit()
        return
      }
    }
  }()

  //  strat go-routine to display status changes in icon/menu
  go func() {
    Logger.Println("Status updater started")
    currentStatus := ""
    currentIcon := 0
    tick := time.NewTimer(333 * time.Millisecond)
    defer tick.Stop()
    for {
      select {
        case yds, ok := <- YD.Updates:
          if ok {
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
          } else {
            Logger.Println("Status updater exited.")
            return
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
