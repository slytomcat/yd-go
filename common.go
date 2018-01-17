package main

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sync"

	"github.com/slytomcat/llog"
)

// notExists returns true when specified path does not exists
func notExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return os.IsNotExist(err)
	}
	return false
}

func expandHome(path string) string {
	if len(path) == 0 || path[0] != '~' {
		return path
	}
	usr, err := user.Current()
	if err != nil {
		llog.Critical("Can't get current user profile:", err)
	}
	return filepath.Join(usr.HomeDir, path[1:])
}

func xdgOpen(uri string) {
	err := exec.Command("xdg-open", uri).Start()
	if err != nil {
		llog.Error(err)
	}
}

func notifySend(icon, title, body string) {
	llog.Debug("Message:", title, ":", body)
	err := exec.Command("notify-send", "-i", icon, title, body).Start()
	if err != nil {
		llog.Error(err)
	}
}

// shortName returns the shorten version of its first parameter. The second parameter specifies
// the maximum number of symbols (runes) in returned string.
func shortName(s string, l int) string {
	r := []rune(s)
	lr := len(r)
	if lr > l {
		b := (l - 3) / 2
		e := b
		if b+e+3 < l {
			e++
		}
		return string(r[:b]) + "..." + string(r[lr-e:])
	}
	return s
}

// LastT type is just map[strig]string protected by RWMutex to be read and updated
// form different goroutines simulationusly
type LastT struct {
	m map[string]*string
	l sync.RWMutex
}

func (l *LastT) reset() {
	l.l.Lock()
	l.m = make(map[string]*string, 10) // 10 - is a maximum lenghth of the last synchronized
	l.l.Unlock()
}

func (l *LastT) update(key, value string) {
	l.l.Lock()
	l.m[key] = &value
	l.l.Unlock()
}

func (l *LastT) get(key string) string {
	l.l.RLock()
	defer l.l.RUnlock()
	return *l.m[key]
}

func (l *LastT) len() int {
	l.l.RLock()
	defer l.l.RUnlock()
	return len(l.m)
}

// func MLC(in string) (out string){
// 	out, ok := Messages[in]
// 	if !ok {
// 		out = in
// 	}
// }
