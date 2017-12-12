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

/* tool function that controls the change of value in variable */
func setChange (v *string, val string, ch *bool) {
  if *v != val {
    *v = val
    *ch = true
  }
}

type YDvals struct {
  stat string      // current status
  prev string      // previous status
  total string     // total space available
  used string      // used space
  trash string     // trash size
  last [10]string  // last-updated files/folders
}

func (val *YDvals) Update(out string) bool {
  changed := false
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
      for i:= 0; i < len(f); i++ {
        if val.last[i] != f[i][1] {
          val.last[i] = f[i][1]
          changed = true
        }
      }
    }
  }
  return changed
}

type YDstat struct {
  update chan string   // input channel for update values with data from strong
  change chan YDvals   // output channel for detected changes
  status chan bool     // input channel for status request
  replay chan string   // output channel for replay on status request
}

func NewYDstatus() YDstat {
  st := YDstat {
    make(chan string),
    make(chan YDvals, 1), // Output shoud be buffered
    make(chan bool),
    make(chan string, 1), // Output shoud be buffered
  }
  go func() {
    yds := YDvals{
        "unknown",
        "unknown",
        "...", "...", "...",
        [10]string{},
      }
    for {
      select {
        case upd := <- st.update:
          if yds.Update(upd) {
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
    NewYDstatus(),
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

  err = watcher.Add(yd.path + "/.sync/cli.log") //"/.sync/status")
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
    log.Println("Daemon started:", string(out))
  }
  if !yd.watcherStat() {
    //log.Println("Watcher not started, start it.")
    yd.watcherStart()
  }
  return "", nil
}

func (yd *YDisk) Stop() (string, error) {
  if yd.getOutput(true) != "" {
    out, err := exec.Command("yandex-disk", "-c", yd.conf, "stop").Output()
    if err != nil {
      log.Fatal(err)
    }
    log.Println("Daemon stopped:", string(out))
  }
  if yd.watcherStat() {
    //log.Println("Watcher was started, stop it.")
    yd.watcherStop()
  }
  return "", nil
}

func (yd *YDisk) Status() string {
  yd.stat.status <- true
  return <- yd.stat.replay
}

func main() {
  // need to check that yandex-disk is installed and properly configured
  // get syncronized path from yandex-disk config
  YD := NewYDisk("/home/stc/.config/yandex-disk/config.cfg", "/home/stc/Yandex.Disk")
  // Start change display routine
  go func() {
    for {
      yds := <- YD.stat.change
      log.Println(strings.Join([]string{"Change detected!\n  Prev = ", yds.prev, "  Stat = ",
                                        yds.stat,"\n  Total=", yds.total, " Used = ",
                                        yds.used, " Trash= ", yds.trash,
                                        "\n  Last =", yds.last[0]},""))
    }
  }()

  log.Println("Status:", YD.Status())
  _, err := YD.Start()
  if err != nil {
    log.Fatal(err)
  }
  log.Println("Daemon Started")

  //time.Sleep(time.Second)
  fmt.Scanln()
  log.Println("Exit requested")
  _, err = YD.Stop()

  time.Sleep(time.Second * 1)
  log.Println("Status", YD.Status())
  log.Println("All done")

}
