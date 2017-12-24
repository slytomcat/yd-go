package main

import (
  "log"
  "os"
  //"time"

  //"fmt"
  //"encoding/json"

  "github.com/slytomcat/YD.go/YDisk"
  . "github.com/slytomcat/YD.go/icons"
  "github.com/slytomcat/systray"

)

/* Initialize default logger */
var Logger *log.Logger = log.New(os.Stderr, "", log.Lshortfile|log.Lmicroseconds) // | log.Lmicroseconds)

func main() {
  systray.Run(onReady, onExit)
}

func onReady() {
  /* TO_DO:
   * 0. Check that daemon is installed (exit if not)
   * 1. Read/create_and_save app configuration file for:
   *  path to daemon config (default "/home/stc/.config/yandex-disk/config.cfg")
   *  icons theme (default "dark")
   *  autostart indicator (default "yes")
   *  start daemon on start (default "yes")
   *  stop daemon on exit (default "no")
   * 2. Read daemon config for:
   *  path to synchronized folder
   *  path to auth file
   * 3. Check that daemon is configured (check auth and conf paths existance, exit if not)
   * */
   // Make systray icon
  systray.SetIcon(IconPause)
  systray.SetTitle("")
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
  //  create new YDisk interface
  // TEST ONLY VALUE should be read from app config
  conf := "/home/stc/.config/yandex-disk/config.cfg"
  // TEST ONLY VALUE should be read from daemon config
  path := "/home/stc/Yandex.Disk"
  YD := YDisk.NewYDisk(conf, path)
  // make go-routine for menu treatment
  go func(){
    for {
      select {
      case <-mStart.ClickedCh:
        YD.Start()
      case <-mStop.ClickedCh:
        YD.Stop()
      case <-mQuit.ClickedCh:
        Logger.Println("Exit requested.")
        YD.Close()
        systray.Quit()
        return
      }
    }
  }()

  //  strat go-routine to display status changes in icon/menu
  go func() {
    Logger.Println("Status updater started")
    //iconsSet := [][]byte{IconBusy1, IconBusy2, IconBusy3, IconBusy4, IconBusy5}
    //animationStop := make(chan bool, 1)
    //iconAnimation := func() {
      //tick := time.NewTicker(333 * time.Millisecond)
      //n := 0
      //for {
        //select {
          //case <- animationStop:
            //tick.Stop()
            //return
          //case <- tick.C:
            ////systray.SetIcon(iconsSet[n])
            //Logger.Println(n)
            //n = (n+1) % 5
          //}
        //}
      //}
    for {
      yds, ok := <- YD.Updates
      if ok {
        mStatus.SetTitle("Status: " + yds.Stat)
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
            systray.SetIcon(IconBusy1)
            //go iconAnimation()
          case "index":
            systray.SetIcon(IconBusy1)
          default:
            systray.SetIcon(IconError)
        }
        if yds.Stat != "none" {
          mStart.Disable()
          mStop.Enable()
        }
        //if yds.Stat != "idle" {
        //  animationStop <- true
        //}
      } else {
        Logger.Println("Status updater exited.")
        return
      }

    }
  }()

}

func onExit() {}
