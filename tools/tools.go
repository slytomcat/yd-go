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
	"sync"
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
	lock   sync.Mutex
	delay  time.Duration
	timer  *time.Timer
	action func()
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
	ds.lock.Lock()
	defer ds.lock.Unlock()
	if ds.timer != nil {
		ds.timer.Stop()
	}
	ds.timer = time.AfterFunc(ds.delay, ds.ActNowIfScheduled)
}

// ActNowIfScheduled triggers an immediate execution of the action if it is scheduled.
// If the action is not scheduled, it does nothing.
// It can be used to execute the scheduled action before timeout on application exit.
func (ds *DelayedActioner) ActNowIfScheduled() {
	ds.lock.Lock()
	defer ds.lock.Unlock()
	if ds.timer != nil {
		ds.timer.Stop()
		ds.timer = nil
		ds.action()
	}
}

// Config is application configuration
type Config struct {
	lock          sync.Mutex       // lock for configuration fields
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
// In case of error it logs the error message but does not return it.
func (c *Config) save() {
	c.lock.Lock()
	data, _ := json.Marshal(c)
	c.lock.Unlock()
	err := os.WriteFile(c.path, data, 0600)
	if err != nil {
		c.log.Warn("can't save config file", "error", err)
	}
}

// SaveChangedNow saves the configuration to the disk immediately if it was changed earlier.
// It can be used to save configuration before application exit without waiting for timeout.
func (c *Config) SaveChangedNow() {
	c.da.ActNowIfScheduled()
}

// Getters and setters for configuration fields. Setters trigger delayed saving of configuration to the disk.

// GetTheme returns the current theme name
func (c *Config) GetTheme() string {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.Theme
}

// SetTheme sets the theme name and triggers delayed saving of configuration to the disk
func (c *Config) SetTheme(theme string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.Theme = theme
	c.da.Act()
}

// GetNotifications returns the current value of Notifications field
func (c *Config) GetNotifications() bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.Notifications
}

// SetNotifications sets the value of Notifications field and triggers delayed saving of configuration to the disk
func (c *Config) SetNotifications(notifications bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.Notifications = notifications
	c.da.Act()
}

// GetStartDaemon returns the current value of StartDaemon field
func (c *Config) GetStartDaemon() bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.StartDaemon
}

// SetStartDaemon sets the value of StartDaemon field and triggers delayed saving of configuration to the disk
func (c *Config) SetStartDaemon(startDaemon bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.StartDaemon = startDaemon
	c.da.Act()
}

// GetStopDaemon returns the current value of StopDaemon field
func (c *Config) GetStopDaemon() bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.StopDaemon
}

// SetStopDaemon sets the value of StopDaemon field and triggers delayed saving of configuration to the disk
func (c *Config) SetStopDaemon(stopDaemon bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.StopDaemon = stopDaemon
	c.da.Act()
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
