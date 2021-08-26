// Package tools contains commonly used functions for yd-go and yd-qgo projects
package tools

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"

	"github.com/slytomcat/llog"
)

// NotExists returns true when specified path does not exists
func NotExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return os.IsNotExist(err)
	}
	return false
}

// XdgOpen opens the uri via xdg-open command
func XdgOpen(uri string) {
	if err := exec.Command("xdg-open", uri).Start(); err != nil {
		llog.Error(err)
	}
}

// ShortName returns the shorten version of its first parameter. The second parameter specifies
// the maximum number of symbols (runes) in returned string.
func ShortName(s string, l int) string {
	r := []rune(s)
	if len(r) < l {
		return s
	}
	b := (l - 3) / 2
	return string(r[:b]) + "..." + string(r[len(r)-(l-3-b):])
}

// Config is applicatinon configuration
type Config struct {
	Conf          string // path to daemon config file
	Theme         string // icons theme name
	Notifications bool   // display desktop notification
	StartDaemon   bool   // start daemon on app start
	StopDaemon    bool
}

// NewConfig returns the application configeration
func NewConfig(cfgFilePath string) *Config {
	cfg := &Config{
		Conf:          os.ExpandEnv("$HOME/.config/yandex-disk/config.cfg"), // path to daemon config file
		Theme:         "dark",                                               // icons theme name
		Notifications: true,                                                 // display desktop notification
		StartDaemon:   true,                                                 // start daemon on app start
		StopDaemon:    false,                                                // stop daemon on app closure
	}

	cfgPath, _ := path.Split(cfgFilePath)
	// Check that app configuration file path exists
	if NotExists(cfgPath) {
		if err := os.MkdirAll(cfgPath, 0766); err != nil {
			llog.Critical("Can't create application configuration path:", err)
		}
	}
	// Check that app configuration file exists
	if NotExists(cfgFilePath) {
		//Create and save new configuration file with default values
		data, _ := json.Marshal(cfg)
		err := os.WriteFile(cfgFilePath, data, 0766)
		if err != nil {
			llog.Critical("Can't save configuration file:", err)
		}
		llog.Debugf("cfg saved: %+v", cfgFilePath)
	} else {
		// Read app configuration file
		data, err := os.ReadFile(cfgFilePath)
		if err != nil {
			llog.Criticalf("reading config file error: %v", err)
		}
		err = json.Unmarshal(data, cfg)
		if err != nil {
			llog.Criticalf("parsing config file error: %v", err)
		}
		llog.Debugf("cfg read: %+v", cfg)
	}
	return cfg
}

// AppInit handles command line arguments and
// initializes logging facility.
// Parameter: appName - name of application,
// Returns: path to config file
func AppInit(appName string, args []string) string {
	var debug bool
	var config string
	f := flag.NewFlagSet(appName, flag.ExitOnError)
	f.BoolVar(&debug, "debug", false, "Allow debugging messages to be sent to stderr")
	f.StringVar(&config, "config", "$HOME/.config/"+appName+"/default.cfg", "Path to the indicator configuration file")
	f.Usage = func() {
		_, _ = fmt.Fprintf(f.Output(), "Usage:\n\n\t\t%q [-debug] [-config=<Path to indicator config>]\n\n", appName)
		f.PrintDefaults()
	}
	_ = f.Parse(args[1:])
	// Initialize logging facility
	llog.SetPrefix("")
	llog.SetFlags(log.Lshortfile | log.Lmicroseconds)
	if debug {
		llog.SetLevel(llog.DEBUG)
		llog.Info("Debugging enabled")
	} else {
		llog.SetLevel(-1)
	}

	return os.ExpandEnv(config)
}
