/*
Package ydisk implements API for yandex-disk daemon. Logging is organized
via github.com/slytomcat/llog package.
*/
package ydisk

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/fsnotify/fsnotify"
)

var log *slog.Logger

// YDvals - Daemon Status structure
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

// A new YDvals constructor
func newYDvals() YDvals {
	return YDvals{
		Stat:   "unknown",
		Prev:   "unknown",
		Total:  "",
		Used:   "",
		Free:   "",
		Trash:  "",
		Last:   []string{},
		ChLast: true,
		Err:    "",
		ErrP:   "",
		Prog:   "",
	}
}

// Tool function that controls the change of value in variable
func setChanged(v *string, val string, c *bool) {
	if *v != val {
		*v = val
		*c = true
	}
}

// update - Updates Daemon status values from the daemon output string.
// Returns true if a change detected in any value, otherwise returns false.
// It uses only strings operation for parsing.
func (val *YDvals) update(out string) bool {
	val.Prev = val.Stat // store previous status but don't track changes of val.Prev
	changed := false    // track changes for values
	if out == "" {
		if setChanged(&val.Stat, "none", &changed); changed {
			val.Total, val.Used, val.Trash, val.Free = "", "", "", ""
			val.Prog, val.Err, val.ErrP, val.ChLast = "", "", "", true
			val.Last = []string{}
		}
		return changed
	}
	n := strings.Index(out, "Last synchronized items:")
	val.ChLast = false // track last list changes separately
	if n > 0 {
		// Parse the "Last synchronized items" section (list of paths and files)
		f := make([]string, 0, 10)
		files := out[n+24:]
		for {
			if p := strings.Index(files, "\n"); p < 0 {
				break
			} else {
				if p > 8 {
					f = append(f, files[strings.Index(files, ":")+3:p-1])
				}
				files = files[p+len("\n"):]
			}
		}
		if len(f) != len(val.Last) {
			val.ChLast = true
			val.Last = f
		} else {
			for i, p := range f {
				setChanged(&val.Last[i], p, &val.ChLast)
			}
		}
	} else { // There is no "Last synchronized items" section
		n = len(out)
		if len(val.Last) > 0 {
			val.Last = []string{}
			val.ChLast = true
		}
	}
	// Parse disk values and status
	// Initialize map with keys that can be missed
	keys := make(map[string]string, 11)
	keys["Sync progress"] = ""
	keys["Error"] = ""
	keys["Path"] = ""
	vals := out[:n]
	for {
		if p := strings.Index(vals, "\n"); p < 0 {
			break
		} else {
			if n := strings.Index(vals[:p], ":"); n > 0 {
				keys[strings.TrimLeftFunc(vals[:n], unicode.IsSpace)] = vals[n+2 : p]
			}
			vals = vals[p+1:]
		}
	}
	for k, v := range keys {
		switch k {
		case "Synchronization core status":
			setChanged(&val.Stat, v, &changed)
		case "Total":
			setChanged(&val.Total, v, &changed)
		case "Used":
			setChanged(&val.Used, v, &changed)
		case "Available":
			setChanged(&val.Free, v, &changed)
		case "Trash size":
			setChanged(&val.Trash, v, &changed)
		case "Sync progress":
			setChanged(&val.Prog, v, &changed)
		case "Error":
			setChanged(&val.Err, v, &changed)
		case "Path":
			if v != "" {
				setChanged(&val.ErrP, v[1:len(v)-1], &changed)
			} else {
				setChanged(&val.ErrP, "", &changed)
			}
		}
	}
	return changed || val.ChLast
}

type watcher struct {
	*fsnotify.Watcher
	active bool // Flag that means that watching path was successfully added
}

func newWatcher() watcher {
	watch, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error("file_watcher", "error", err)
		os.Exit(1)
	}
	return watcher{
		Watcher: watch,
		active:  false,
	}
}

func (w *watcher) activate(path string) {
	if !w.active {
		path := filepath.Join(path, ".sync/cli.log")
		err := w.Add(path)
		if err != nil {
			log.Debug("file_watcher", "path", path, "error", err)
			return
		}
		log.Debug("file_watcher", "status", "added")
		w.active = true
	}
}

// YDisk provides methods to interact with yandex-disk (methods: Start, Stop, Output), path
// of synchronized catalogue (property Path) and channel for receiving yandex-disk status
// changes (property Changes).
type YDisk struct {
	Path     string        // Path to synchronized folder (obtained from yandex-disk conf. file)
	Changes  chan YDvals   // Output channel for detected changes in daemon status
	conf     string        // Path to yandex-disc configuration file
	exe      string        // Path to yandex-disk executable
	exit     chan struct{} // Stop signal/replay channel for Event handler routine
	activate func()        // Function to activate watcher after daemon creation
}

// NewYDisk creates new YDisk structure for communication with yandex-disk daemon
// Parameter:
//
//	conf - full path to yandex-disk daemon configuration file
//
// Checks performed in the beginning:
//
//   - check that yandex-disk was installed
//   - check that yandex-disk was properly configured
//
// When something not good NewYDisk returns not nil error
func NewYDisk(conf string, logger *slog.Logger) (*YDisk, error) {
	log = logger
	exe, path, err := checkDaemon(conf)
	if err != nil {
		return nil, err
	}
	watch := newWatcher()
	log.Debug("yandex-disk", "executable", exe)
	yd := YDisk{
		Path:     path,
		Changes:  make(chan YDvals, 1), // Output should be buffered
		conf:     conf,
		exe:      exe,
		exit:     make(chan struct{}),
		activate: func() { watch.activate(path) },
	}
	// start event handler in separate goroutine
	go yd.eventHandler(watch)
	// Try to activate watching at the beginning. It may fail but it is not a problem
	// as it can be activated later (on Start of daemon).
	yd.activate()
	log.Debug("YDisk", "status", "initialized", "path", path)
	return &yd, nil
}

// eventHandler works in separate goroutine until YDisk.exit channel receives a bool value (any).
func (yd *YDisk) eventHandler(watch watcher) {
	log.Debug("daemon_event_handler", "status", "started")
	yds := newYDvals()
	interval := 1
	tick := time.NewTimer(time.Millisecond * 100) // First time trigger it quickly to update the current status
	defer func() {
		watch.Close()
		tick.Stop()
		close(yd.Changes)
		log.Debug("daemon_event_handler", "status", "exited")
		yd.exit <- struct{}{} // Report exit completion
	}()
	var source string
	for {
		select {
		case err := <-watch.Errors:
			log.Error("file_watcher", "error", err)
			return
		case <-yd.exit:
			return
		case <-watch.Events:
			source = "watcher"
			interval = 1
		case <-tick.C:
			source = fmt.Sprintf("timer%ds", interval)
			if yds.Stat == "busy" || yds.Stat == "index" {
				interval = 2 // keep 2s interval in busy mode
			} else {
				if interval < 32 {
					interval <<= 1 // continuously increase timer interval: 2s, 4s, 8s.
				}
			}
		}
		// in both cases (Timer or Watcher events):
		//  - check for daemon changes and send changed values in case of change
		if yds.update(yd.getOutput(false)) {
			log.Debug("change", "source", source, "prev", yds.Prev, "new", yds.Stat,
				"S", len(yds.Total) > 0, "L", len(yds.Last), "E", len(yds.Err) > 0)
			yd.Changes <- yds
			// in case of any change reset the timer interval
			interval = 1
		}
		//  - restart timer when daemon is running
		if yds.Stat != "none" {
			tick.Reset(time.Duration(interval) * time.Second)
		} else {
			tick.Stop()
		}
	}
}

func (yd YDisk) getOutput(userLang bool) string {
	cmd := []string{yd.exe, "status", "-c", yd.conf}
	if !userLang {
		cmd = append([]string{"env", "-i", "TEMP=" + os.TempDir()}, cmd...)
	}
	out, err := exec.Command(cmd[0], cmd[1:]...).Output()
	if err != nil {
		if message := strings.TrimSuffix(string(out), "\n"); message != "Error: daemon not started" {
			log.Error("daemon_status", "error", err.Error(), "message", message)
		}
		return ""
	}
	return string(out)
}

// Close deactivates the daemon connection: stops event handler that closes file watcher
// and Changes channel.
func (yd *YDisk) Close() {
	yd.exit <- struct{}{}
	<-yd.exit // Wait for the event handler completion
}

// Output returns the output string of `yandex-disk status` command in the current user language.
func (yd *YDisk) Output() string {
	return yd.getOutput(true)
}

// Start runs `yandex-disk start` if daemon was not started before.
func (yd *YDisk) Start() error {
	if yd.getOutput(true) == "" {
		out, err := exec.Command(yd.exe, "start", "-c", yd.conf).Output()
		if err != nil {
			log.Error("daemon_start", "error", err)
			return err
		}
		log.Debug("daemon_start", "message", string(bytes.TrimRight(out, " \n")))
	} else {
		log.Debug("daemon_start", "status", "already_started")
	}
	yd.activate() // try to activate watching after daemon start. It shouldn't fail on started daemon
	return nil
}

// Stop runs `yandex-disk stop` if daemon was not stopped before.
func (yd *YDisk) Stop() error {
	if yd.getOutput(true) != "" {
		out, err := exec.Command(yd.exe, "stop", "-c", yd.conf).Output()
		if err != nil {
			log.Error("daemon stop", "error", err)
			return err
		}
		log.Debug("daemon_stop", "message", string(bytes.TrimRight(out, " \n")))
	} else {
		log.Debug("daemon_stop", "status", "already_stopped")
	}
	return nil
}
