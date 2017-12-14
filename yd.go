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
  "os"
  "encoding/json"
)

/* Initialize logger */
var lg *log.Logger = log.New(os.Stderr, "", log.Lmicroseconds | log.Lshortfile)

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

/* string representation of []string slice */
func list(Last []string) string {
  l := []string{}
  for _, s := range(Last) {
    if s != "" {
      l = append(l, s)
    }
  }
  return strings.Join(l, ",")
}

/* Daemon Status values */
type YDvals struct {
  Stat string      // current Status
  Prev string      // Previous Status
  Total string     // Total space available
  Used string      // Used space
  Trash string     // Trash size
  Last [10]string  // Last-updated files/folders
}

func newYDvals() YDvals {
  return YDvals{
        "unknown",
        "unknown",
        "", "", "",
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
  val.Prev = val.Stat
  if out == "" {
    setChange(&val.Stat, "none", &changed)
    if changed {
      val.Total = ""
      val.Used = ""
      val.Trash = ""
      val.Last = [10]string{}
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
          setChange(&val.Stat, v[2], &changed)
        case "Total" :
          setChange(&val.Total, v[2], &changed)
        case "Used" :
          setChange(&val.Used, v[2], &changed)
        case "Trash" :
          setChange(&val.Trash, v[2], &changed)
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
        setChange(&val.Last[i], p, &changed)
      }
    }
  }
  return changed
}

/* Status control component */
type YDstat struct {
  update chan string   // input channel for update values with data from the daemon output string
  change chan YDvals   // output channel for detected changes
  status chan bool     // input channel for status request
  replay chan string   // output channel for replay on status request
}

/* This control component implemented as State-full go-routine with 4 communication channels */
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
            lg.Println(strings.Join([]string{"Change detected!\n  Prev=", yds.Prev, "  Stat=", yds.Stat,
                    "\n  Total=", yds.Total, " Used=", yds.Used, " Trash=", yds.Trash,
                    "\n  Last=", list(yds.Last[:])},""))
          }
        case stat := <- st.status:
          switch stat {
            case true:       // true : Full state request
              st.replay <- yds.Stat
            case false:      // false : report status and exit
              st.replay <- yds.Stat
              lg.Print("Status component routine finished")
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
  watch uint32    // Watcher Status (0 - not started) !!! Use atomic functions to access it!
}

func NewYDisk(conf string, path string) YDisk {
  lg.Println("New YDisk created.\n  Conf:", conf, "\n  Path:", path)
  yd := YDisk{
    conf,
    path,
    newYDstatus(),
    make(chan bool, 1),
    0,
  }
  yd.watcherStart()
  return yd
}

func (yd YDisk) getOutput(userLang bool) (string) {
  cmd := []string{ "yandex-disk", "-c", yd.conf, "status"}
  if !userLang {
    cmd = append([]string{"env", "-i", "LANG='en_US.UTF8'"}, cmd...)
  }
  //lg.Printf("cmd=", cmd)
  out, err := exec.Command(cmd[0], cmd[1:]...).Output()
  //lg.Printf("Status=%s", string(out))
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
    lg.Fatal(err)
  }
  tick := time.NewTimer(time.Second)
  n := 0
  atomic.StoreUint32(&yd.watch, 1)
  lg.Println("File watcher started")

  go func() {
    defer func() {
      tick.Stop()
      watcher.Close()
      atomic.StoreUint32(&yd.watch, 0)
      lg.Println("File watcher routine finished")
    }()
    for {
      select {
        case <-watcher.Events: //event := <-watcher.Events:
          //lg.Println("Watcher event:", event)
          tick.Reset(time.Second)
          n = 0
          yd.stat.update <- yd.getOutput(false)
        case err := <-watcher.Errors:
          lg.Println("Watcher error:", err)
          return
        case <-tick.C:
          //lg.Println("timer:", n)
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
    lg.Fatal(err)
  }
  lg.Println("Watch path added")
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
      lg.Fatal(err)
    }
    lg.Println("Daemon start:", string(out))
  }
  if !yd.watcherStat() {
    //lg.Println("Watcher not started, start it.")
    yd.watcherStart()
  }
  lg.Println("Daemon Started")
  return "", nil
}

func (yd *YDisk) Stop() (string, error) {
  if yd.getOutput(true) != "" {
    out, err := exec.Command("yandex-disk", "-c", yd.conf, "stop").Output()
    if err != nil {
      lg.Fatal(err)
    }
    lg.Println("Daemon stop:", string(out))
  }
  //if yd.watcherStat() {
  //  //lg.Println("Watcher was started, stop it.")
  //  yd.watcherStop()
  // }
  lg.Println("Daemon Stopped")
  return "", nil
}

func (yd *YDisk) Status() string {
  yd.stat.status <- true
  return <- yd.stat.replay
}

func (yd *YDisk) Close() {
  if yd.watcherStat() {
    yd.watcherStop()
  }
  yd.stat.status <- false
  time.Sleep(time.Millisecond * 100)
}

func notify(msg string) {
  err := exec.Command("notify-send", msg).Run()
  if err != nil {
    lg.Fatal(err)
  }
}

func CommandCycle(YD *YDisk) {
  // command receive cycle
  for {
    lg.Print("Commands: start, stop, status, exit")
    inp:=""
    fmt.Scanln(&inp)
    switch inp {
      case "start":
        if _, err := YD.Start(); err != nil { lg.Fatal(err) }
      case "stop":
        if _, err := YD.Stop(); err != nil { lg.Fatal(err) }
      case "status":
        lg.Println("Current status:", YD.Status())
      case "exit":
        lg.Println("Exit requested")
        return
    }
  }
}

func main() {
  // TO_DO:
  // 1. need to check that yandex-disk is installed and properly configured
  // 2. get synchronized path from yandex-disk config
  YD := NewYDisk("/home/stc/.config/yandex-disk/config.cfg", "/home/stc/Yandex.Disk")
  lg.Println("Current status:", YD.Status())

  // TO_DO:
  // 1. Decide what to do with status updates:
  //  - how to show them to user
  //  - if show facility is in the oter program - how to pass updates to that process (pipe?/socket?)

  // Start the change display routine - it just stub to see updates in the log
  exit := make(chan bool)
  go func() {
    for {
      select{
        case yds := <- YD.stat.change:
          msj, _ := json.Marshal(yds)
          notify(string(msj))
        case <- exit:
          lg.Print("Status display routine finished")
          return
      }
    }
  }()

  // TO_DO:
  // 1. Check that yandex-disk should be started on startup
  // 2. Call YD.Start() only it is requered

  CommandCycle(&YD)

  // TO_DO:
  // 1. Check that yandex-disk should be stopped on exit
  // 2. Call YD.Stop() only it is requered
  lg.Println("Exit Status:", YD.Status())
  exit <- true
  YD.Close()
  lg.Println("All done. Bye!")

}
