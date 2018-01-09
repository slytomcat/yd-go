package YDisk

/* This package provides API for yandex-disk daemon.
 * A new daemon connection is created with call of:
 *  NewYDisk (conf, path string) YDisk
 * Parameters:
 *  conf - path to yandex-disk configuration file (by default it is: ~/.config/yandex-disk/config.cfg)
 *  path - path to user folder that is synchronized by daemon (default path: ~/Yandex.Disk)
 * Returns new YDisk structure with the following items:
 *  Changes chan YDvals - output channel that provides all detected changes in daemon status within
 *                        the structure YDvals (see description of YDvals structure)
 * The daemon connection has following methods:
 *  Start()          - starts the daemon with the specified configuration
 *  Stop()           - stops the daemon with the specified configuration
 *  Output() string  - returns the daemon status message (in the current user Language)
 *  Close()          - closes the daemon connection (stops all service routines and file watcher)
 *
 * YDvals structure has following items:
 *  Stat string      - Current Status
 *  Prev string      - Previous Status
 *  Total string     - Total space available
 *  Used string      - Used space
 *  Free string      - Free space
 *  Trash string     - Trash size
 *  Last []string    - Last-updated files/folders list (10 or less items)
 *  ChLast bool      - Indicator that Last was changed
 *  Err string       - Error status message
 *  ErrP string      - Error path
 *  Prog string      - Synchronization progress (when in busy status)
 *
 * Debugging messages can be obtained via log that must be *log.log type. By default it
 * prints the messages to os.Stderr.
 * */

import (
	"log"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

var AllDone sync.WaitGroup

/* Daemon Status values */
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

func newyDvals() YDvals {
	return YDvals{
		"unknown",      // Current Status
		"unknown",      // Previous Status
		"", "", "", "", // Total, Used, Free, Trash
		[]string{},			// Last
		false,					// ChLast
		"", "", "",			// Err, ErrP, Prog
	}
}

/* Tool function that controls the change of value in variable */
func setChange(v *string, val string, ch *bool) {
	if *v != val {
		*v = val
		*ch = true
	}
}

/* Update Daemon status values from the daemon output string
 * Returns true if change detected in any value, otherwise returns false */
func (val *YDvals) update(out string) bool {
	val.Prev = val.Stat // store previous status but don't track changes of val.Prev
	changed := false    // track changes for values
	errChanged := false // track changes for error
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
	keys := map[string]string{"Sync": "", "Error": "", "Path": ""}
	// Take only first word in the phrase before ":"
	for _, s := range regexp.MustCompile(`\s*([^ ]+).*: (.*)`).FindAllStringSubmatch(split[0], -1) {
		if s[2][0] == byte('\'') {
			s[2] = s[2][1 : len(s[2])-1] // remove ' in the begging and at end
		}
		keys[s[1]] = s[2]
	}
	// map representation of switch_case clause
	for k, v := range map[string]*string{"Synchronization": &val.Stat,
		"Total":     &val.Total,
		"Used":      &val.Used,
		"Available": &val.Free,
		"Trash":     &val.Trash,
		"Error":     &val.Err,
		"Path":      &val.ErrP,
		"Sync":      &val.Prog,
	} {
		setChange(v, keys[k], &changed)
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
				setChange(&val.Last[i], p[1], &val.ChLast)
			}
		}
	} else { // len(split) = 1 - there is no section with last sync. paths
		if len(val.Last) > 0 {
			val.Last = []string{}
			val.ChLast = true
		}
	}
	return changed || val.ChLast || errChanged
}

/* Status control component */
type yDstatus struct {
	updates chan string // input channel for update values with data from the daemon output string
	Changes chan YDvals // output channel for detected changes or daemon output
}

/* This control component implemented as State-full go-routine with 3 communication channels */
func newyDstatus() yDstatus {
	st := yDstatus{
		make(chan string),
		make(chan YDvals, 1), // Output should be buffered
	}
	go func() {
		log.Println("Status component started")
		AllDone.Add(1)
		defer AllDone.Done()
		yds := newyDvals()
		for {
			upd, ok := <-st.updates
			if ok {
				if yds.update(upd) {
					log.Println("Change: P=", yds.Prev, "S=", yds.Stat,
						"T=", yds.Total, "L=", len(yds.Last), "E=", yds.Err)
					st.Changes <- yds
				}
			} else { // st.updates channel closed - exit
				close(st.Changes)
				log.Println("Status component routine finished")
				return
			}
		}
	}()
	return st
}

type watcher struct {
	watch  *fsnotify.Watcher
	path   bool // Flag that means that watching path was successfully added
	Events chan fsnotify.Event
	Errors chan error
}

func newwatcher() watcher {
	watch, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	return watcher{
		watch,
		false,
		watch.Events,
		watch.Errors,
	}
}

func (w *watcher) activate(path string) {
	if !w.path {
		err := w.watch.Add(filepath.Join(path, ".sync/cli.log"))
		if err != nil {
			log.Println("Watch path error:", err)
			return
		}
		log.Println("Watch path added")
		w.path = true
	}
}

func (w *watcher) close() {
	// TO_DO_Maybe: path need to removed before close watcher?
	w.watch.Close()
}

type YDisk struct {
	conf    string      // Path to yandex-disc configuration file
	path    string      // Path to synchronized folder (should be obtained from y-d conf. file)
	stat    yDstatus    // Status object
	watch   watcher     // Watcher object
	exit    chan bool   // Stop signal for Event handler routine
	Changes chan YDvals // Transfered from status component Updates channel
}

func NewYDisk(conf, path string) YDisk {
	// Requirements:
	// 1. yandex-disk have to be installed and properly configured
	// 2. path to configuration and synchronized paths from yandex-disk conf-file have to be
	//    provided in arguments
	stat := newyDstatus()
	yd := YDisk{
		conf,
		path,
		stat,
		newwatcher(),
		make(chan bool),
		stat.Changes,
	}
	yd.watch.activate(yd.path) // Try to activate watching at the beginning. It can fail

	go func() {
		log.Println("Event handler started")
		AllDone.Add(1)
		tick := time.NewTimer(time.Millisecond * 500)
		interval := 2
		defer func() {
			tick.Stop()
			log.Println("Event handler routine finished")
			AllDone.Done()
		}()
		busy_status := false
		out := ""
		for {
			select {
			case <-yd.watch.Events: //event := <-yd.watch.Events:
				//log.Println("Watcher event:", event)
				tick.Reset(time.Millisecond * 500)
				interval = 2
			case <-tick.C:
				//log.Println("Timer interval:", interval)
				if busy_status {
					interval = 2 // keep 2s interval in busy mode
				}
				if interval < 10 {
					tick.Reset(time.Duration(interval) * time.Second)
					interval <<= 1 // continuously increase timer interval: 2s, 4s, 8s.
				}
			case err := <-yd.watch.Errors:
				log.Println("Watcher error:", err)
				return
			case <-yd.exit:
				return
			}
			out = yd.getOutput(false)
			busy_status = strings.HasPrefix(out, "Sync progress")
			yd.stat.updates <- out
		}
	}()

	log.Println("New YDisk created and initialized.\n  Conf:", conf, "\n  Path:", path)
	return yd
}

func (yd *YDisk) Close() {
	yd.exit <- true
	yd.watch.close()
	close(yd.stat.updates)
	AllDone.Wait()
	log.Println("All done. Bye!")
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

// Request function
func (yd *YDisk) Output() string {
	return yd.getOutput(true)
}

// Commands
func (yd *YDisk) Start() {
	if yd.getOutput(true) == "" {
		out, err := exec.Command("yandex-disk", "-c", yd.conf, "start").Output()
		if err != nil {
			log.Println(err)
		}
		log.Println("Daemon start:", string(out))
	} else {
		log.Println("Daemon already Started")
	}
	yd.watch.activate(yd.path) // try to activate watching after daemon start. It shouldn't fail
}

func (yd *YDisk) Stop() {
	if yd.getOutput(true) != "" {
		out, err := exec.Command("yandex-disk", "-c", yd.conf, "stop").Output()
		if err != nil {
			log.Println(err)
		}
		log.Println("Daemon stop:", string(out))
		return
	}
	log.Println("Daemon already stopped")
}
