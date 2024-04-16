// Package tools contains commonly used functions for yd-go and yd-qgo projects
package tools

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"strings"
)

var llog *slog.Logger

// NotExists returns true when specified path does not exists
func NotExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return errors.Is(err, fs.ErrNotExist)
	}
	return false
}

// XdgOpen opens the uri via xdg-open command
func XdgOpen(uri string) error {
	if err := exec.Command("xdg-open", uri).Start(); err != nil {
		return err
	}
	return nil
}

// MakeTitle returns the shorten version of its first parameter. The second parameter specifies
// the maximum number of symbols (runes) in returned string. It also replaces underscore symbol with
// the special unicode symbols sequence that looks very similar to the original underscore
func MakeTitle(s string, l int) string {
	r := []rune(s)
	if len(r) < l {
		return replaceUnderscore(s)
	}
	b := (l - 3) / 2
	return replaceUnderscore(string(r[:b])) + "..." + replaceUnderscore(string(r[len(r)-(l-3-b):]))
}

// replaceUnderscore replaces underscore (special symbol for menu shortcut) to special unicode symbols which looks like original underscore
func replaceUnderscore(s string) string {
	return strings.ReplaceAll(s, "_", "\u2009\u0332\u2009") // thin space + combining low line + thin space
}

// Config is application configuration
type Config struct {
	path          string // config file path
	Conf          string // path to daemon config file
	Theme         string // icons theme name
	Notifications bool   // display desktop notification
	StartDaemon   bool   // start daemon on app start
	StopDaemon    bool   // stop daemon on app exit
}

// NewConfig returns the application configuration
func NewConfig(cfgFilePath string) (*Config, error) {
	cfg := &Config{
		path: cfgFilePath, // store path for Save method
		// fill it with default values
		Conf:          os.ExpandEnv("$HOME/.config/yandex-disk/config.cfg"), // path to daemon config file
		Theme:         "dark",                                               // icons theme name
		Notifications: true,                                                 // display desktop notification
		StartDaemon:   true,                                                 // start daemon on app start
		StopDaemon:    false,                                                // stop daemon on app closure
	}

	cfgPath, _ := path.Split(cfgFilePath)
	// Check that app configuration file path exists
	if NotExists(cfgPath) {
		if err := os.MkdirAll(cfgPath, 0700); err != nil {
			return nil, fmt.Errorf("Can't create application configuration path: %v", err)
		}
	}
	// Check that app configuration file exists
	if NotExists(cfgFilePath) {
		//Create and save new configuration file with default values
		err := cfg.Save()
		if err != nil {
			return nil, fmt.Errorf("default config saving error: %v", err)
		}
	} else {
		// Read app configuration file
		data, err := os.ReadFile(cfgFilePath)
		if err != nil {
			return nil, fmt.Errorf("reading config file error: %v", err)
		}
		err = json.Unmarshal(data, cfg)
		if err != nil {
			return nil, fmt.Errorf("parsing config file error: %v", err)
		}
	}
	return cfg, nil
}

// Save stores application configuration to the disk
func (c *Config) Save() error {
	data, _ := json.Marshal(c)
	err := os.WriteFile(c.path, data, 0664)
	if err != nil {
		return fmt.Errorf("Can't save configuration file: %v", err)
	}
	return nil
}

// AppInit handles command line arguments and
// initializes logging facility.
// Parameter: appName - name of application,
// Returns: path to config file
func AppInit(appName string, args []string, version string) (string, bool) {
	var pv bool
	var debug bool
	var config string
	f := flag.NewFlagSet(appName, flag.ExitOnError)
	f.BoolVar(&debug, "debug", false, "Allow debugging messages to be sent to stderr")
	f.StringVar(&config, "config", "$HOME/.config/"+appName+"/default.cfg", "Path to the indicator configuration file")
	f.BoolVar(&pv, "version", false, "Print out version information and exit")
	f.Usage = func() {
		_, _ = fmt.Fprintf(f.Output(), "%s\nUsage:\n\n\t\t%q [-debug] [-config=<Path to indicator config>]\n\n", getVersion(appName, version), appName)
		f.PrintDefaults()
	}
	_ = f.Parse(args[1:])
	if pv {
		fmt.Print(getVersion(appName, version))
		os.Exit(0)
	}
	return os.ExpandEnv(config), debug
}

func getVersion(appName, version string) string {
	return fmt.Sprintf("%s ver.: %s\n", appName, version)
}
