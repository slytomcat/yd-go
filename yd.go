package main

import (
  "log"
  "fmt"
  "time"
  "github.com/fsnotify/fsnotify"
  "os/exec"
  "regexp"
  "strings"
  "sync/atomic"
  //"os"
)

/* Tool function that returns shorten version (up to l symbols) of original string  */
func ShortName(f string, l int) string {
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

/* Daemon status values */
type YDvals struct {
  stat string      // current status
  prev string      // previous status
  total string     // total space available
  used string      // used space
  trash string     // trash size
  last [10]string  // last-updated files/folders
}

func newYDvals() YDvals {
  return YDvals{
        "unknown",
        "unknown",
        "...", "...", "...",
        [10]string{},
      }
}

/* Tool function that controls the change of value in variable */
func setChange (v *string, val string, ch *bool) {
  if *v != val {
    *v = val
    *ch = true
  }
}

/* Update Daemon status values from the daemon output string
 * Returns true if values change detected otherways returns false */
func (val *YDvals) update(out string) bool {
  changed := false  // track changes
  val.prev = val.stat
  if out == "" {
    setChange(&val.stat, "none", &changed)
    if changed {
      val.total = "..."
      val.used = "..."
      val.trash = "..."
      val.last = [10]string{}
    }
  } else {
    split := strings.Split(string(out), "Last synchronized items:")
    vals := regexp.MustCompile(`\s*([^ ]+).*: (.*)`).FindAllStringSubmatch(split[0], -1)
    for _, v := range vals {
      if v[2][0] == byte('\'') {
        v[2] = v[2][1:len(v[2])-1]
      }
      switch v[1] {
        case "Synchronization" :
          setChange(&val.stat, v[2], &changed)
        case "Total" :
          setChange(&val.total, v[2], &changed)
        case "Used" :
          setChange(&val.used, v[2], &changed)
        case "Trash" :
          setChange(&val.trash, v[2], &changed)
      }
    }
    if len(split) > 1 {
      f := regexp.MustCompile(`: '(.*).\n`).FindAllStringSubmatch(split[1], -1)
      var p string
      for i:= 0; i < 10; i++ {
        if i < len(f) {
          p = f[i][1]
        } else {
          p = ""
        }
        setChange(&val.last[i], p, &changed)
      }
    }
  }
  return changed
}

/* Status control component */
type YDstat struct {
  update chan string   // input channel for update values with data from string
  change chan YDvals   // output channel for detected changes
  status chan bool     // input channel for status request
  replay chan string   // output channel for replay on status request
}

func newYDstatus() YDstat {
  st := YDstat {
    make(chan string),
    make(chan YDvals, 1), // Output should be buffered
    make(chan bool),
    make(chan string, 1), // Output should be buffered
  }
  go func() {
    yds := newYDvals()
    for {
      select {
        case upd := <- st.update:
          if yds.update(upd) {
            st.change <- yds
          }
        case stat := <- st.status:
          if stat {  // true : status request
            st.replay <- yds.stat
          } else {   // false : exit
            return
          }
      }
    }
  }()
  return st
}

type YDisk struct {
  conf string     // Path to yandex-disc configuration file
  path string     // Path to synchronized folder (should be obtained from y-d conf. file)
  stat YDstat     // Status object
  stop chan bool  // Stop signal channel
  watch uint32    // Watcher status (0 - not started) !!! Use atomic functions to access it!
}

func NewYDisk(conf string, path string) YDisk {
  log.Println("New YDisk created.\n  Conf:", conf, "\n  Path:", path)
  return YDisk{
    conf,
    path,
    newYDstatus(),
    make(chan bool, 1),
    0,
  }
}

func (yd YDisk) getOutput(userLang bool) (string) {
  cmd := []string{ "yandex-disk", "-c", yd.conf, "status"}
  if !userLang {
    cmd = append([]string{"env", "-i", "LANG='en_US.UTF8'"}, cmd...)
  }
  //log.Printf("cmd=", cmd)
  out, err := exec.Command(cmd[0], cmd[1:]...).Output()
  //log.Printf("Status=%s", string(out))
  if err != nil {
    out = []byte{}
  }
  return string(out)
}

func (yd YDisk) Output() string {
  return yd.getOutput(true)
}

func (yd *YDisk) watcherStart() {
  const second = int(time.Second)
  watcher, err := fsnotify.NewWatcher()
  if err != nil {
    log.Fatal(err)
  }

  go func() {
    tick := time.NewTimer(time.Second)
    n := 0
    atomic.StoreUint32(&yd.watch, 1)
    log.Println("Watcher started")
    defer func() {
      tick.Stop()
      watcher.Close()
      atomic.StoreUint32(&yd.watch, 0)
      log.Println("Watcher stopped")
    }()
    for {
      select {
        case <-watcher.Events: //event := <-watcher.Events:
          //log.Println("Watcher event:", event)
          tick.Reset(time.Second)
          n = 0
          yd.stat.update <- yd.getOutput(false)
        case err := <-watcher.Errors:
          log.Println("Watcher error:", err)
          return
        case <-tick.C:
          //log.Println("timer:", n)
          // continiously increase timer period: 2s, 4s, 8s.
          if n < 4 {
            n++
            tick.Reset(time.Duration(second * n * 2))
          }
          yd.stat.update <- yd.getOutput(false)
        case <-yd.stop:
          return
      }
    }
  }()

  err = watcher.Add(yd.path + "/.sync/cli.log") // TO_DO: make path via library function
  if err != nil {
    log.Fatal(err)
  }
  log.Println("Watch path added")
}

func (yd *YDisk) watcherStop() {
  yd.stop<-true
}

func (yd *YDisk) watcherStat() bool {
  return atomic.LoadUint32(&yd.watch) != 0
}

func (yd *YDisk) Start() (string, error) {
  if yd.getOutput(true) == "" {
    out, err := exec.Command("yandex-disk", "-c", yd.conf, "start").Output()
    if err != nil {
      log.Fatal(err)
    }
    log.Println("Daemon start:", string(out))
  }
  if !yd.watcherStat() {
    //log.Println("Watcher not started, start it.")
    yd.watcherStart()
  }
  log.Println("Daemon Started")
  return "", nil
}

func (yd *YDisk) Stop() (string, error) {
  if yd.getOutput(true) != "" {
    out, err := exec.Command("yandex-disk", "-c", yd.conf, "stop").Output()
    if err != nil {
      log.Fatal(err)
    }
    log.Println("Daemon stop:", string(out))
  }
  if yd.watcherStat() {
    //log.Println("Watcher was started, stop it.")
    yd.watcherStop()
  }
  log.Println("Daemon Stopped")
  return "", nil
}

func (yd *YDisk) Status() string {
  yd.stat.status <- true
  return <- yd.stat.replay
}

func main() {
  // TO_DO:
  // 1. need to check that yandex-disk is installed and properly configured
  // 2. get synchronized path from yandex-disk config
  YD := NewYDisk("/home/stc/.config/yandex-disk/config.cfg", "/home/stc/Yandex.Disk")

  // TO_DO:
  // 1. Decide what to do with status updates:
  //  - how to show them to user
  //  - if show facility is in the oter program - how to pass updates to that process (pipe?/socket?)

  // Start the change display routine - it just stub to see updates in the log
  go func() {
    for {
      yds := <- YD.stat.change
      log.Println(strings.Join([]string{"Change detected!\n  Prev = ", yds.prev, "  Stat = ",
                                        yds.stat,"\n  Total=", yds.total, " Used = ",
                                        yds.used, " Trash= ", yds.trash,
                                        "\n  Last =", yds.last[0]},""))
    }
  }()

  log.Println("Current status:", YD.Status())

  // TO_DO:
  // 1. Check that yandex-disk should be started on startup
  // 2. Call YD.Start() only it is requered
  _, err := YD.Start()
  if err != nil {
    log.Fatal(err)
  }

  //time.Sleep(time.Second)
  fmt.Scanln()
  log.Println("Current status:", YD.Status())
  log.Println("Exit requested")

  // TO_DO:
  // 1. Check that yandex-disk should be stopped on exit
  // 2. Call YD.Stop() only it is requered
  _, err = YD.Stop()

  time.Sleep(time.Second * 1)
  log.Println("Current status:", YD.Status())
  log.Println("All done. Bye!")

}
