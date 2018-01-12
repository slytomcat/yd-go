package ydisk

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/slytomcat/llog"
)

func notExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return os.IsNotExist(err)
	}
	return false
}

// checkDaemon checks that yandex-disk daemon is installed.
// It reads the provided daemon configuration file and checks existence of synchronized folder
// and authorization file ('passwd' file). If one of them is not exists then checkDaemon exits
// from programm.
// It returns the user catalog that is synchronized by daemon in case of success check.
func checkDaemon(conf string) string {
	if notExists("/usr/bin/yandex-disk") {
		llog.Critical("Yandex.Disk CLI utility is not installed. Install it first.")
	}
	f, err := os.Open(conf)
	if err != nil {
		llog.Critical("Daemon configuration file opening error:", err)
	}
	defer f.Close()
	reader := io.Reader(f)
	line := ""
	dir := ""
	auth := ""
	for {
		n, err := fmt.Fscanln(reader, &line)
		if n == 0 {
			break
		}
		if err != nil {
			llog.Error(err)
		}
		if strings.HasPrefix(line, "dir") {
			dir = line[5 : len(line)-1]
		}
		if strings.HasPrefix(line, "auth") {
			auth = line[6 : len(line)-1]
		}
		if dir != "" && auth != "" {
			break
		}
	}
	if notExists(dir) || notExists(auth) {
		llog.Critical("Daemon is not configured. Run:\nyandex-disk setup")
	}
	return dir
}
