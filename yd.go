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
var lg *log.Logger = log.New(os.Stderr, "", log.Lshortfile) // | log.Lmicroseconds)

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
  Free string      // Free space
  Trash string     // Trash size
  Last []string    // Last-updated files/folders
  ChLast bool      // Indicator that Last was changed
  Err string       // Error status messaage
  ErrP string      // Error path
  Prog string      // Syncronization progress (when in busy status)
}

func newYDvals() YDvals {
  return YDvals{
        "unknown",
        "unknown",
        "", "", "", "", // Total, Used, Free, Trash
        []string{},     // Last
        false,          // ChLast
        "", "", "",     // Err, ErrP, Prog
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
 * Returns true if change detected in any value, otherways returns false */
func (val *YDvals) update(out string) bool {
  val.Prev = val.Stat  // store previous status but don't track changes of val.Prev
  changed := false     // track changes for values
  if out == "" {
    setChange(&val.Stat, "none", &changed)
    if changed {
      val.Total, val.Used, val.Trash, val.Free = "", "", "", ""
      val.Prog, val.Err, val.ErrP, val.ChLast = "", "", "", true
      val.Last = []string{}
      return true
    }
    return false
  }
  split := strings.Split(string(out), "Last synchronized items:")
  // Need to remove "Path to " as another "Path:" exists in case of access error
  split[0] = strings.Replace(split[0], "Path to ", "", 1)
  // Take only first word in the phrase before ":"
  for _, v := range regexp.MustCompile(`\s*([^ ]+).*: (.*)`).FindAllStringSubmatch(split[0], -1) {
    if v[2][0] == byte('\'') {
      v[2] = v[2][1:len(v[2])-1]
    }
    switch v[1] {
      case "Synchronization" :
        switch v[2] {
          case "no internet access":
            v[2] = "no_net"
          case "index":
            v[2] = "busy"
        }
        setChange(&val.Stat, v[2], &changed)
      case "Sync" :
        setChange(&val.Prog, v[2], &changed)
      case "Total" :
        setChange(&val.Total, v[2], &changed)
      case "Used" :
        setChange(&val.Used, v[2], &changed)
      case "Available" :
        setChange(&val.Free, v[2], &changed)
      case "Trash" :
        setChange(&val.Trash, v[2], &changed)
      case "Error" :
        setChange(&val.Err, v[2], &changed)
      case "Path" :
        setChange(&val.ErrP, v[2], &changed)
    }
  }

  // Parse the "Last syncronized items" section (list of paths and files)
  val.ChLast = false  // track last list changes separately
  if len(split) > 1 {
    f := regexp.MustCompile(`: '(.*)'\n`).FindAllStringSubmatch(split[1], -1)
    if len(f) != len(val.Last) {
      val.ChLast = true
      val.Last = []string{}
      for _, p := range f {
        val.Last = append(val.Last, p[1])
      }
    } else {  // len(split) = 1 - there is no section with last sync. paths
      for i, p := range f {
        setChange(&val.Last[i], p[1], &val.ChLast)
      }
    }
  } else {
    if len(val.Last) > 0 {
      val.Last = []string{}
      val.ChLast = true
    }
  }
  return changed || val.ChLast
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
            lg.Println("Change: Prev=", yds.Prev, "Stat=", yds.Stat,
                       "Total=", yds.Total, "Len(Last)=", len(yds.Last), "Err=", yds.Err)
          }
        case stat := <- st.status:
          switch stat {
            case true:       // true : status request
              st.replay <- yds.Stat
            case false:      // false : exit
              lg.Println("Status component routine finished")
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
  stop chan bool  // Stop signal channel file wathcer routine
  watch uint32    // Watcher status (0 - not started) !!! Use atomic functions to access it!
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
  //lg.Println("cmd=", cmd)
  out, err := exec.Command(cmd[0], cmd[1:]...).Output()
  //lg.Println("Status=%s", string(out))
  if err != nil {
    out = []byte{}
  }
  return string(out)
}

func (yd YDisk) Output() string {
  return yd.getOutput(true)
}

func (yd *YDisk) watcherStart() {
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
          tick.Reset(time.Millisecond * 500)
          n = 0
          yd.stat.update <- yd.getOutput(false)
        case err := <-watcher.Errors:
          lg.Println("Watcher error:", err)
          return
        case <-tick.C:
          // continiously increase timer period: 2s, 4s, 8s.
          if n < 4 {
            n++
            tick.Reset(time.Duration(n * 2) * time.Second)
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

func (yd *YDisk) Start() {
  if yd.getOutput(true) == "" {
    out, err := exec.Command("yandex-disk", "-c", yd.conf, "start").Output()
    if err != nil {
      lg.Println(err)
    }
    lg.Println("Daemon start:", string(out))
  } else {
    lg.Println("Daemon already Started")
  }
  if !yd.watcherStat() {
    yd.watcherStart()
  }

}

func (yd *YDisk) Stop() {
  if yd.getOutput(true) != "" {
    out, err := exec.Command("yandex-disk", "-c", yd.conf, "stop").Output()
    if err != nil {
      lg.Println(err)
    }
    lg.Println("Daemon stop:", string(out))
    return
  }
  lg.Println("Daemon already stopped")
}

func (yd *YDisk) Sync() {
  if yd.getOutput(true) != "" {
    out, err := exec.Command("yandex-disk", "-c", yd.conf, "sync").Output()
    if err != nil {
      lg.Fatal(err)
    }
    lg.Println("Sync:", string(out))
    return
  }
  lg.Println("Sync can't be inicialized")
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
}

func notify(msg string) {
  err := exec.Command("notify-send", msg).Run()
  if err != nil {
    lg.Fatal(err)
  }
}

// Command receive cycle
func CommandCycle(YD *YDisk) {
  var inp string
  for {
    //fmt.Println("Commands: start, stop, sync, status, output, exit")
    fmt.Scanln(&inp)
    switch inp {
      case "start":
        YD.Start()
      case "stop":
        YD.Stop()
      case "sync":
        YD.Sync()
      case "status":
        fmt.Printf("{\"Status\": \"%s\"}\n", YD.Status())
      case "output":
        msj, _ := json.Marshal(YD.Output())
        fmt.Printf("{\"Output\": %s}\n", string(msj))
      case "exit":
        lg.Println("Exit requested")
        return
    }
  }
}

func main() {
  if len(os.Args) < 3 {
    lg.Fatal("Error: Path to yandex-disc config-file and path to synchronized folder",
             "must be provided via first and second command line arguments")
  }
  // TO_DO:
  // 1. need to check that yandex-disk is installed and properly configured
  // 2. get synchronized path from yandex-disk config
  // or
  // pass paths via command line arguments
  YD := NewYDisk(os.Args[1], os.Args[2])
  //YD := NewYDisk("/home/stc/.config/yandex-disk/config.cfg", "/home/stc/Yandex.Disk")
  lg.Println("Current status:", YD.Status())

  // TO_DO:
  // 1. Decide what to do with status updates:
  //  - how to show them to user = via external program
  //  - if show facility is in the other program - how to pass
  //  updates to that process (pipe?/socket?) = stdout - updates, stdin - commands, stderr - log

  // Start the change display routine
  exit := make(chan bool)
  go func() {
    for {
      select{
        case yds := <- YD.stat.change:
          msj, _ := json.Marshal(yds)
          //notify(string(msj))
          fmt.Println(string(msj))
        case <- exit:
          lg.Println("Status display routine finished")
          return
      }
    }
  }()

  // TO_DO:
  // 1. Check that yandex-disk should be started on startup
  // 2. Call YD.Start() only it is requered
  //  or
  // Leave the solution on external program

  CommandCycle(&YD)

  // TO_DO:
  // 1. Check that yandex-disk should be stopped on exit
  // 2. Call YD.Stop() only it is requered
  //  or
  // Leave the solution on external program

  lg.Println("Exit Status:", YD.Status())
  exit <- true
  YD.Close()
  time.Sleep(time.Millisecond * 50)
  lg.Println("All done. Bye!")

}
