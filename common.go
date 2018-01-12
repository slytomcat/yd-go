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
		llog.Debug(err)
	}
}

func notifySend(icon, title, body string) {
	err := exec.Command("notify-send", "-i", icon, title, body).Start()
	if err != nil {
		llog.Debug(err)
	}
}

// shortName returns the shorten version of its first parameter. The second parameter specifies
// the maximum number of symbols (runes) in returned string.
func shortName(f string, l int) string {
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

type LastT struct {
	m map[string]string
	l sync.RWMutex
}

func (l *LastT) reset() {
	l.l.Lock()
	l.m = make(map[string]string, 10)
	l.l.Unlock()
}

func (l *LastT) update(key, value string) {
	l.l.Lock()
	l.m[key] = value
	l.l.Unlock()
}

func (l LastT) get(key string) string {
	l.l.RLock()
	defer l.l.RUnlock()
	return l.m[key]
}
