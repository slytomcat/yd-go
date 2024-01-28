package ydisk

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
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
// from program.
// It returns the user catalogue that is synchronized by daemon in case of success check.
func checkDaemon(conf string) (string, string, error) {
	exe, err := exec.LookPath("yandex-disk")
	if err != nil {
		msg := "Yandex.Disk CLI utility is not installed. Install it first"
		llog.Error(msg)
		return "", "", fmt.Errorf(msg)
	}
	f, err := os.Open(conf)
	if err != nil {
		llog.Error("Daemon configuration file opening error:", err)
		return "", "", err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	var line, dir, auth string
	for {
		line, err = reader.ReadString('\n')
		if err != nil {
			break
		}
		if strings.HasPrefix(line, "dir") {
			dir = line[5 : len(line)-2]
		}
		if strings.HasPrefix(line, "auth") {
			auth = line[6 : len(line)-2]
		}
		if dir != "" && auth != "" {
			break
		}
	}
	if err != nil && err != io.EOF {
		return "", "", err
	}
	if notExists(dir) || notExists(auth) {
		msg := "Daemon is not configured. First run: `yandex-disk setup`"
		llog.Error(msg)
		return "", "", fmt.Errorf("%s", msg)
	}
	return exe, dir, nil
}
