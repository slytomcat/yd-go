package main

import (
  "log"
  "fmt"
  "time"
  "github.com/fsnotify/fsnotify"
  "os/exec"
  "regexp"
  "strings"
  "os"
  "encoding/json"
)

/* Initialize default logger */
var Logger *log.Logger = log.New(os.Stderr, "", log.Lshortfile) // | log.Lmicroseconds)

/* Daemon Status values */
type yDvals struct {
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

func newyDvals() yDvals {
  return yDvals{
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
func (val *yDvals) update(out string) bool {
  val.Prev = val.Stat  // store previous status but don't track changes of val.Prev
  changed := false     // track changes for values
  if out == "" {
    setChange(&val.Stat, "none", &changed)
    if changed {
      val.Total, val.Used, val.Trash, val.Free = "", "", "", ""
      val.Prog, val.Err, val.ErrP, val.ChLast = "", "", "", true
      val.Last = []string{}
    }
    return changed
  }
  split := strings.Split(string(out), "Last synchronized items:")
  // Need to remove "Path to " as another "Path:" exists in case of access error
  split[0] = strings.Replace(split[0], "Path to ", "", 1)
  // Initialize map with keys that can be missed
  keys := map[string]string {"Sync":"", "Error":"", "Path":""}
  // Take only first word in the phrase before ":"
  for _, s := range regexp.MustCompile(`\s*([^ ]+).*: (.*)`).FindAllStringSubmatch(split[0], -1) {
    if s[2][0] == byte('\'') {
      s[2] = s[2][1:len(s[2])-1]   // remove ' in the beggining and at end
    }
    keys[s[1]] = s[2]
  }
  for k, v := range keys {
    switch k {
      case "Synchronization":
        setChange(&val.Stat, v, &changed)
      case "Sync":
        setChange(&val.Prog, v, &changed)
      case "Total":
        setChange(&val.Total, v, &changed)
      case "Used":
        setChange(&val.Used, v, &changed)
      case "Available":
        setChange(&val.Free, v, &changed)
      case "Trash":
        setChange(&val.Trash, v, &changed)
      case "Error":
        setChange(&val.Err, v, &changed)
      case "Path":
        setChange(&val.ErrP, v, &changed)
    }
  }

  // Parse the "Last synchronized items" section (list of paths and files)
  val.ChLast = false  // track last list changes separately
  if len(split) > 1 {
    f := regexp.MustCompile(`: '(.*)'\n`).FindAllStringSubmatch(split[1], -1)
    if len(f) != len(val.Last) {
      val.ChLast = true
      val.Last = []string{}
      for _, p := range f {
        val.Last = append(val.Last, p[1])
      }
    } else {
      for i, p := range f {
        setChange(&val.Last[i], p[1], &val.ChLast)
      }
    }
  } else {   // len(split) = 1 - there is no section with last sync. paths
    if len(val.Last) > 0 {
      val.Last = []string{}
      val.ChLast = true
    }
  }
  return changed || val.ChLast
}

/* Status control component */
type yDstatus struct {
  Update chan string   // input channel for update values with data from the daemon output string
  Change chan yDvals   // output channel for detected changes
  exit chan bool       // input channel for exit request
}

/* This control component implemented as State-full go-routine with 4 communication channels */
func newyDstatus() yDstatus {
  st := yDstatus {
    make(chan string),
    make(chan yDvals, 1), // Output should be buffered
    make(chan bool),
  }
  go func() {
    yds := newyDvals()
    for {
      select {
        case upd := <- st.Update:
          if yds.update(upd) {
            Logger.Println("Change: Prev=", yds.Prev, "Stat=", yds.Stat,
                       "Total=", yds.Total, "Len(Last)=", len(yds.Last), "Err=", yds.Err)
            st.Change <- yds
          }
        case <- st.exit:
          Logger.Println("Status component routine finished")
          return
      }
    }
  }()
  return st
}

type watcher struct {
  watch *fsnotify.Watcher
  path bool        // Flag that means that wather path was succesfully added
  Events chan fsnotify.Event
  Errors chan error
}

func newwatcher() watcher {
  watch, err := fsnotify.NewWatcher()
  if err != nil {
    Logger.Fatal(err)
  }
  w := watcher{
      watch,
      false,
      watch.Events,
      watch.Errors,
    }
  return w
}

func (w *watcher) activate(path string) {
  if !w.path {
    err := w.watch.Add(path + "/.sync/cli.log") // TO_DO: make path via library function
    if err != nil {
      Logger.Println("Watch path error:", err)
      return
    }
    Logger.Println("Watch path added")
    w.path = true
  }
}

func (w *watcher) close() {
  // TO_DO_Maybe: path need to removed before close watcher?
  w.watch.Close()
}

type YDisk struct {
  conf string     // Path to yandex-disc configuration file
  path string     // Path to synchronized folder (should be obtained from y-d conf. file)
  stat yDstatus   // Status object
  watch watcher   // Watcher object
  exit chan bool  // Stop signal for Event handler routine
  Commands chan string // Input channel for commands
  Updates chan string  // Output channel for status updates
}

func NewYDisk(conf string, path string) YDisk {
  // Requerements:
  // 1. yandex-disk have to be installed and properly configured
  // 2. path to config and synchronized path from yandex-disk config have to be provided in arguments
  yd := YDisk{
    conf,
    path,
    newyDstatus(),
    newwatcher(),
    make(chan bool),
    make(chan string),
    make(chan string),
  }
  yd.watch.activate(yd.path)  // Try to activate wathing at the beggining

  tick := time.NewTimer(time.Millisecond * 500)
  interval := 2
  Logger.Println("Event handler started")

  go func() {
    defer func() {
      tick.Stop()
      Logger.Println("Event handler routine finished")
    }()
    busy_status := false
    out := ""
    for {
      select {
        case <-yd.watch.Events: //event := <-yd.watch.watch.Events:
          //Logger.Println("Watcher event:", event)
          tick.Reset(time.Millisecond * 500)
          interval = 2
        case <-tick.C:
          //Logger.Println("Timer interval:", interval)
          if busy_status {
            interval = 2  // keep 2s interval in busy mode
          }
          if interval < 10 {
            tick.Reset(time.Duration(interval) * time.Second)
            interval<<=1 // continuously increase timer interval: 2s, 4s, 8s.
          }
        case err := <-yd.watch.Errors:
          Logger.Println("Watcher error:", err)
          return
        case <-yd.exit:
          return
      }
      out = yd.getOutput(false)
      busy_status = strings.HasPrefix(out, "Sync progress")
      yd.stat.Update <- out
    }
  }()

  // Activate command handler and output formatter
  go func() {
    var msj []byte
    for {
      select {
        case cmd := <- yd.Commands:
          switch cmd {
            case "start":
              yd.start()
            case "stop":
              yd.stop()
            case "output":
              msj, _ = json.Marshal(yd.getOutput(true))
              yd.Updates <- "{\"Output\": " + string(msj) + "}"
            //case "sync":
            //  yd.sync()
            case "exit":
              yd.Close()
              Logger.Println("Command handler routine finished")
              return
          }
        case yds := <- yd.stat.Change:
          msj, _ = json.Marshal(yds)
          yd.Updates <- string(msj)
      }
    }
  }()

  Logger.Println("New YDisk created.\n  Conf:", conf, "\n  Path:", path)
  return yd
}

func (yd YDisk) getOutput(userLang bool) (string) {
  cmd := []string{ "yandex-disk", "-c", yd.conf, "status"}
  if !userLang {
    cmd = append([]string{"env", "-i", "LANG='en_US.UTF8'"}, cmd...)
  }
  //Logger.Println("cmd=", cmd)
  out, err := exec.Command(cmd[0], cmd[1:]...).Output()
  //Logger.Println("Status=%s", string(out))
  if err != nil {
    return ""
  }
  return string(out)
}

func (yd *YDisk) start() {
  if yd.getOutput(true) == "" {
    out, err := exec.Command("yandex-disk", "-c", yd.conf, "start").Output()
    if err != nil {
      Logger.Println(err)
    }
    Logger.Println("Daemon start:", string(out))
  } else {
    Logger.Println("Daemon already Started")
  }
  yd.watch.activate(yd.path)   // try to activate watching afret daemon start
}

func (yd *YDisk) stop() {
  if yd.getOutput(true) != "" {
    out, err := exec.Command("yandex-disk", "-c", yd.conf, "stop").Output()
    if err != nil {
      Logger.Println(err)
    }
    Logger.Println("Daemon stop:", string(out))
    return
  }
  Logger.Println("Daemon already stopped")
}

func (yd *YDisk) Close() {
  yd.exit <- true
  yd.watch.close()
  yd.stat.exit <- false
}

func main() {
  if len(os.Args) < 3 {
    Logger.Fatal("Error: Path to yandex-disc config-file and path to synchronized folder",
             "must be provided via first and second command line arguments")
  }
  YD := NewYDisk(os.Args[1], os.Args[2])
  //YD := NewYDisk("/home/stc/.config/yandex-disk/config.cfg", "/home/stc/Yandex.Disk")

  // stdin -> Commads
  // Status updates -> stdout
  // Log messages -> stderr
  // External program/operator have to decide what to do with daemon and pass command.
  // Wrapper itself doesn't auto-start or stop daemo on its start/exit

  // Start the change display routine
  exit := make(chan bool)
  go func() {
    for {
      select{
        case out:= <- YD.Updates:
          fmt.Println(out)
        case <- exit:
          Logger.Println("Output display routine finished")
          return
      }
    }
  }()

  // stdin reader cycle
  var inp string
  for inp != "exit" {
    //fmt.Println("Commands: start, stop, output, exit")
    inp = ""
    fmt.Scanln(&inp)
    YD.Commands <- inp
  }

  exit <- true

  Logger.Println("Exit requested.")
  time.Sleep(time.Millisecond * 50)
  Logger.Println("All done. Bye!")

 }
