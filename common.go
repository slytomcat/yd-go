package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
)

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
		log.Fatal("Can't get current user profile:", err)
	}
	return filepath.Join(usr.HomeDir, path[1:])
}

func xdgOpen(uri string) {
	err := exec.Command("xdg-open", uri).Start()
	if err != nil {
		log.Println(err)
	}
}

func notifySend(icon, title, body string) {
	err := exec.Command("notify-send", "-i", icon, title, body).Start()
	if err != nil {
		log.Println(err)
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

var AppConfigFile string

func init() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Allow debugging messages to be sent to stderr")
	flag.StringVar(&AppConfigFile, "config", "~/.config/yd-go/default.cfg", "Path to the indicator configuration file")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n\n\t\tyd-go [-debug] [-config=<Path to indicator config>]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	/* Initialize logging facility */
	if debug {
		log.SetOutput(os.Stderr)
		log.SetPrefix("")
		log.SetFlags(log.Lshortfile | log.Lmicroseconds)
		log.Println("Debugging enabled")
	} else {
		log.SetOutput(ioutil.Discard)
	}
}
