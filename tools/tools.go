// Package tools contains commonly used functions for yd-go and yd-qgo projects
package tools

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/slytomcat/confjson"
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
	lr := len(r)
	if lr > l {
		b := (l - 3) / 2
		return string(r[:b]) + "..." + string(r[lr - (l - b - 3):])
	}
	return s
}

// AppInit handles command line arguments, loads the application configuration and
// initializes logging facility. Parameter:
//   appName - name of application,
// Returns *map[string]interface{} - with application configuration
func AppInit(appName string) map[string]interface{} {
	var debug bool
	var AppConfigFile string
	flag.BoolVar(&debug, "debug", false, "Allow debugging messages to be sent to stderr")
	flag.StringVar(&AppConfigFile, "config", "$HOME/.config/"+appName+"/default.cfg", "Path to the indicator configuration file")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n\n\t\t"+appName+" [-debug] [-config=<Path to indicator config>]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	// Initialize logging facility
	llog.SetOutput(os.Stderr)
	llog.SetPrefix("")
	llog.SetFlags(log.Lshortfile | log.Lmicroseconds)
	if debug {
		llog.SetLevel(llog.DEBUG)
		llog.Info("Debugging enabled")
	} else {
		llog.SetLevel(-1)
	}

	// Prepare the application configuration
	// Make default app configuration
	AppCfg := map[string]interface{}{
		"Conf":          os.ExpandEnv("$HOME/.config/yandex-disk/config.cfg"), // path to daemon config file
		"Theme":         "dark",                                               // icons theme name
		"Notifications": true,                                                 // display desktop notification
		"StartDaemon":   true,                                                 // start daemon on app start
		"StopDaemon":    false,                                                // stop daemon on app closure
	}
	// Check that app configuration file path exists
	AppConfigHome := os.ExpandEnv("$HOME/.config/" + appName)
	if NotExists(AppConfigHome) {
		if err := os.MkdirAll(AppConfigHome, 0766) ;err != nil {
			llog.Critical("Can't create application configuration path:", err)
		}
	}
	// Path to app configuration file path always comes from command-line flag
	AppConfigFile = os.ExpandEnv(AppConfigFile)
	llog.Debug("Configuration:", AppConfigFile)
	// Check that app configuration file exists
	if NotExists(AppConfigFile) {
		//Create and save new configuration file with default values
		if err := confjson.Save(AppConfigFile, AppCfg); err != nil {
			llog.Critical("Can't create application configuration file:", err)
		}
	} else {
		// Read app configuration file
		AppCfg, err := confjson.Load(AppConfigFile)
		if err != nil {
			llog.Critical("Can't access application configuration file:", err)
		}
	}
	return AppCfg
}
