/*
Package ydisk implements API for yandex-disk daemon. Logging is organized
via github.com/slytomcat/llog package.
*/
package ydisk

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/slytomcat/llog"
)

// YDvals - Daemon Status values
type YDvals struct {
	Stat   string   // Current Status
	Prev   string   // Previous Status
	Total  string   // Total space available
	Used   string   // Used space
	Free   string   // Free space
	Trash  string   // Trash size
	Last   []string // Last-updated files/folders list (10 or less items)
	ChLast bool     // Indicator that Last was changed
	Err    string   // Error status message
	ErrP   string   // Error path
	Prog   string   // Synchronization progress (when in busy status)
}

func newYDvals() YDvals {
	return YDvals{
		"unknown",      // Current Status
		"unknown",      // Previous Status
		"", "", "", "", // Total, Used, Free, Trash
		[]string{}, // Last
		false,      // ChLast
		"", "", "", // Err, ErrP, Prog
	}
}

/* Tool function that controls the change of value in variable */
func setChanged(v *string, val string, c *bool) {
	if *v != val {
		*v = val
		*c = true
	}
}

// update updates Daemon status values from the daemon output string.
// Returns true if a change detected in any value, otherwise returns false */
func (val *YDvals) update(out string) bool {
	val.Prev = val.Stat // store previous status but don't track changes of val.Prev
	changed := false    // track changes for values
	if out == "" {
		setChanged(&val.Stat, "none", &changed)
		if changed {
			val.Total, val.Used, val.Trash, val.Free = "", "", "", ""
			val.Prog, val.Err, val.ErrP, val.ChLast = "", "", "", true
			val.Last = []string{}
		}
		return changed
	}
	split := strings.Split(out, "Last synchronized items:")
	// Need to remove "Path to " as another "Path:" exists in case of access error
	split[0] = strings.Replace(split[0], "Path to ", "", 1)
	// Initialize map with keys that can be missed
	keys := map[string]string{"Sync": "", "Error": "", "Path": ""}
	// Take only first word in the phrase before ":"
	for _, s := range regexp.MustCompile(`\s*([^ ]+).*: (.*)`).FindAllStringSubmatch(split[0], -1) {
		if s[2][0] == byte('\'') {
			s[2] = s[2][1 : len(s[2])-1] // remove ' in the begging and at end
		}
		keys[s[1]] = s[2]
	}
	// map representation of switch_case clause
	for k, v := range map[string]*string{
		"Synchronization": &val.Stat,
		"Total":           &val.Total,
		"Used":            &val.Used,
		"Available":       &val.Free,
		"Trash":           &val.Trash,
		"Error":           &val.Err,
		"Path":            &val.ErrP,
		"Sync":            &val.Prog,
	} {
		setChanged(v, keys[k], &changed)
	}
	// Parse the "Last synchronized items" section (list of paths and files)
	val.ChLast = false // track last list changes separately
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
				setChanged(&val.Last[i], p[1], &val.ChLast)
			}
		}
	} else { // len(split) = 1 - there is no section with last sync. paths
		if len(val.Last) > 0 {
			val.Last = []string{}
			val.ChLast = true
		}
	}
	return changed || val.ChLast
}

type watcher struct {
	*fsnotify.Watcher
	active bool // Flag that means that watching path was successfully added
}

func newwatcher() watcher {
	watch, err := fsnotify.NewWatcher()
	if err != nil {
		llog.Critical(err)
	}
	return watcher{
		watch,
		false,
	}
}

func (w *watcher) activate(path string) {
	if !w.active {
		err := w.Add(filepath.Join(path, ".sync/cli.log"))
		if err != nil {
			llog.Debug("Watch path error:", err)
			return
		}
		llog.Debug("Watch path added")
		w.active = true
	}
}

// YDisk provides methods to interact with yandex-disk (methods: Start, Stop, Output), path
// of synchronized catalog (property Path) and channel for receiving yandex-disk status
// changes (property Changes).
type YDisk struct {
	Path     string      // Path to synchronized folder (obtained from yandex-disk conf. file)
	Changes  chan YDvals // Output channel for detected changes in daemon status
	conf     string      // Path to yandex-disc configuration file
	exit     chan bool   // Stop signal/replay chanel for Event handler routine
	activate func()      // Function to activate watcher after start of daemon
}

// NewYDisk creates new YDisk structure for communication with yandex-disk daemon
// Parameter:
//  conf - full path to yandex-disk daemon configuration file
//
// Checks performed in the beginning:
//
//  - check that yandex-disk has installed
//  - check that yandex-disk was properly configured
//
// When something not good NewYDisk raise panic
func NewYDisk(conf string) YDisk {
	path := checkDaemon(conf)
	watch := newwatcher()
	yd := YDisk{
		path,
		make(chan YDvals, 1), // Output should be buffered
		conf,
		make(chan bool),
		func() { watch.activate(path) },
	}
	// start event handler in separate goroutine
	go yd.eventHandler(watch)
	yd.activate() // Try to activate watching at the beginning. It may fail
	llog.Debug("New YDisk created and initialized.\n  Conf:", conf, "\n  Path:", path)
	return yd
}

// eventHandler works in separate goroutine untill YDisk.exit channel receives a bool value (any).
func (yd *YDisk) eventHandler(watch watcher) {
	llog.Debug("Event handler started")
	yds := newYDvals()
	tick := time.NewTimer(time.Millisecond * 100)  // First time trigger it quickly to update icon and meny 
	interval := 1
	defer func() {
		watch.Close()
		tick.Stop()
		close(yd.Changes)
		llog.Debug("Event handler exited")
		yd.exit <- true // Report exit completion
	}()
	for {
		select {
		case err := <-watch.Errors:
			llog.Error("Watcher error:", err)
			return
		case <-yd.exit:
			return
		case event := <-watch.Events:
			llog.Debug("Watcher event:", event)
			tick.Reset(time.Second)
			interval = 1
		case <-tick.C:
			llog.Debug("Timer interval:", interval)
			if yds.Stat == "busy" || yds.Stat == "index" {
				interval = 2 // keep 2s interval in busy mode
			} else {
				interval <<= 1 // continuously increase timer interval: 2s, 4s, 8s.
			}
			if interval < 10 {
				tick.Reset(time.Duration(interval) * time.Second)
			}
		}
		if yds.update(yd.getOutput(false)) {
			llog.Debug("Change: ", yds.Prev, ">", yds.Stat,
				"S", len(yds.Total) > 0, "L", len(yds.Last), "E", len(yds.Err) > 0)
			yd.Changes <- yds
		}
		//llog.Debug("Event processed")
	}
}

func (yd YDisk) getOutput(userLang bool) string {
	cmd := []string{"yandex-disk", "-c", yd.conf, "status"}
	if !userLang {
		cmd = append([]string{"env", "-i", "LANG='en_US.UTF8'"}, cmd...)
	}
	out, err := exec.Command(cmd[0], cmd[1:]...).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// Close deactivates the daemon connection: stops event handler that closes file watcher
// and Changes channel.
func (yd *YDisk) Close() {
	yd.exit <- true
	<-yd.exit // Wait for the event handler completion
}

// Output returns the output string of `yandex-disk status` command in the current user language.
func (yd *YDisk) Output() string {
	return yd.getOutput(true)
}

// Start runs `yandex-disk start` if daemon was not started before.
func (yd *YDisk) Start() {
	if yd.getOutput(true) == "" {
		out, err := exec.Command("yandex-disk", "-c", yd.conf, "start").Output()
		if err != nil {
			llog.Error(err)
		}
		llog.Debugf("Daemon start: %s", bytes.TrimRight(out, " \n"))
	} else {
		llog.Debug("Daemon already started")
	}
	yd.activate() // try to activate watching after daemon start. It shouldn't fail
}

// Stop runs `yandex-disk stop` if daemon was not stopped before.
func (yd *YDisk) Stop() {
	if yd.getOutput(true) != "" {
		out, err := exec.Command("yandex-disk", "-c", yd.conf, "stop").Output()
		if err != nil {
			llog.Error(err)
		}
		llog.Debugf("Daemon stop: %s", bytes.TrimRight(out, " \n"))
		return
	}
	llog.Debug("Daemon already stopped")
}
