// Package tools contains commonly used functions for yd-go and yd-qgo projects
package tools

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync/atomic"
	"time"
)

var (
	xdgOpenCmd = "xdg-open"
)

// NotExists returns true when specified path does not exists
func NotExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return errors.Is(err, fs.ErrNotExist)
	}
	return false
}

// XdgOpen opens the uri via xdg-open command
func XdgOpen(uri string) error {
	return exec.Command(xdgOpenCmd, uri).Start()
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

// DelayedActioner is a helper struct for perform actions with delay.
// It allows to avoid multiple actions when action called several times in a short period of time.
type DelayedActioner struct {
	delay     time.Duration
	timer     *time.Timer
	scheduled atomic.Bool
	action    func()
}

// NewDelayedActioner returns new DelayedActioner with specified action and delay
func NewDelayedActioner(action func(), delay time.Duration) *DelayedActioner {
	return &DelayedActioner{
		delay:  delay,
		action: action,
	}
}

// Act tries to perform the action after timeout.
// If next call of Act happens before the previous timeout, the timeout will start from beginning.
func (ds *DelayedActioner) Act() {
	if !ds.scheduled.CompareAndSwap(false, true) {
		ds.timer.Stop() // stop previous timer
	}
	ds.timer = time.AfterFunc(ds.delay, ds.action)
}

// ActNow triggers an immediate execution of the action.
// It can be used to execute the action before application exit without waiting for timeout.
func (ds *DelayedActioner) ActNow() {
	if ds.scheduled.CompareAndSwap(true, false) {
		ds.timer.Stop() // stop previous timer
		ds.action()
	}
}

// Config is application configuration
type Config struct {
	path          string           // path to configuration file
	da            *DelayedActioner // delayed actioner for saving configuration to the disk
	log           *slog.Logger     // logger for logging configuration saving errors
	Conf          string           // path to daemon config file
	Theme         string           // icons theme name
	Notifications bool             // display desktop notification
	StartDaemon   bool             // start daemon on app start
	StopDaemon    bool             // stop daemon on app exit
}

// NewConfig returns the application configuration
func NewConfig(cfgFilePath string, delay time.Duration, log *slog.Logger) (*Config, error) {
	cfg := &Config{
		path: cfgFilePath,
		log:  log,
		// fill it with default values
		Conf:          os.ExpandEnv("$HOME/.config/yandex-disk/config.cfg"), // path to daemon config file
		Theme:         "dark",                                               // icons theme name
		Notifications: true,                                                 // display desktop notification
		StartDaemon:   true,                                                 // start daemon on app start
		StopDaemon:    false,                                                // stop daemon on app closure
	}
	cfg.da = NewDelayedActioner(cfg.save, delay)

	cfgPath, _ := path.Split(cfgFilePath)
	if cfgPath == "" {
		cfgPath = "." // if no path is specified, use current directory
	} else if NotExists(cfgPath) {
		if err := os.MkdirAll(cfgPath, 0700); err != nil {
			return nil, fmt.Errorf("can't create application configuration path: %v", err)
		}
	}
	// Check that the configuration file is exists
	if NotExists(cfgFilePath) {
		// Try to save new configuration file with default values
		cfg.save()
	} else {
		// Read the configuration file
		data, err := os.ReadFile(cfgFilePath)
		if err != nil {
			return nil, fmt.Errorf("reading config file error: %v", err)
		}
		if len(data) == 0 { // empty file
			cfg.save()      // try to save default config to the file
			return cfg, nil // return default config
		}
		err = json.Unmarshal(data, cfg)
		if err != nil {
			return nil, fmt.Errorf("parsing config file error: %v", err)
		}
		if cfg.Theme != "dark" && cfg.Theme != "light" {
			return nil, fmt.Errorf("wrong theme name: '%s' (should be 'dark' or 'light')", cfg.Theme)
		}
	}
	return cfg, nil
}

// save writes the configuration to the disk. It is used as action for DelayedActioner and should not be called directly.
func (c *Config) save() {
	data, _ := json.Marshal(c)
	err := os.WriteFile(c.path, data, 0600)
	if err != nil {
		c.log.Warn("can't save config file", "error", err)
	}
}

// Save tries to store application configuration to the disk after timeout.
// If next call of Save happens before the previous timeout, the timeout will start from beginning.
// In case of error it only logs the warning message.
func (c *Config) Save() {
	c.da.Act()
}

// SaveChangedNow saves the configuration to the disk immediately if it was changed earlier.
// It can be used to save configuration before application exit without waiting for timeout.
func (c *Config) SaveChangedNow() {
	c.da.ActNow()
}

// SetupLogger initializes the logger for application
func SetupLogger(debug bool, out io.Writer) *slog.Logger {
	// set logging level
	logLevel := new(slog.LevelVar)
	if debug {
		logLevel.Set(slog.LevelDebug)
	} else {
		logLevel.Set(slog.LevelInfo)
	}
	return slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: logLevel}))
}

// GetParams read the command line parameters and returns configuration file path and boolean value for debug logging activation.
// When app is called with -h or -version or with wrong option it will call os.Exit().
func GetParams(appName string, args []string, version string) (string, bool) {
	var pv bool
	var debug bool
	var config string
	f := flag.NewFlagSet(appName, flag.ExitOnError)
	f.BoolVar(&debug, "debug", false, "Allow debugging messages to be sent to stdout")
	f.StringVar(&config, "config", "$HOME/.config/"+appName+"/default.cfg", "Path to the indicator configuration file")
	f.BoolVar(&pv, "version", false, "Print out version information and exit")
	f.Usage = func() {
		_, _ = fmt.Fprintf(f.Output(), "%s\nUsage:\n\n\t%s [-debug] [-config=<Path to indicator config>] [-version]\n\n", getVersion(appName, version), appName)
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
