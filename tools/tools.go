package tools

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sync"

	"github.com/slytomcat/llog"
)

// NotExists returns true when specified path does not exists
func NotExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return os.IsNotExist(err)
	}
	return false
}

var userHome string
var once sync.Once
// ExpandHome returns full path expanding ~ as $HOME 
func ExpandHome(path string) string {
	if len(path) == 0 || path[0] != '~' {
		return path
	}
	once.Do(func() {
		usr, err := user.Current()
		if err != nil {
			llog.Critical("Can't get current user profile:", err)
		}
		userHome = usr.HomeDir
		llog.Debug("User home folder:", userHome)
	})
	return filepath.Join(userHome, path[1:])
}

// XdgOpen opens the uri via xdg-open command
func XdgOpen(uri string) {
	err := exec.Command("xdg-open", uri).Start()
	if err != nil {
		llog.Error(err)
	}
}

// ShortName returns the shorten version of its first parameter. The second parameter specifies
// the maximum number of symbols (runes) in returned string.
func ShortName(s string, l int) string {
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

