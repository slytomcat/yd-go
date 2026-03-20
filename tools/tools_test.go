package tools

import (
	"io"
	"os"
	"path"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeTitle(t *testing.T) {
	tests := []struct {
		in  string
		out string
		l   int
	}{
		{in: "1234567890", out: "1234567890", l: 20},
		{in: "1234567890", out: "12...890", l: 8},
		{in: "1234567890123", out: "12...123", l: 8},
		{in: "русский текст", out: "русский текст", l: 20},
		{in: "русский текст дада", out: "рус...дада", l: 10},
		{in: "one_two", out: "one\u2009\u0332\u2009two", l: 10},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.out, MakeTitle(tc.in, tc.l))
	}
}

func TestNotExists(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.False(t, NotExists(wd))
	require.True(t, NotExists("/Unreal path+*$"))
}

// makeTempCfgFile creates temporary config file with specified content and returns its path. If content is nil, the file will not be created,
// but the path will be returned.
func makeTempCfgFile(t *testing.T, content *string) string {
	file := path.Join(t.TempDir(), "default.cfg")
	if content != nil {
		if err := os.WriteFile(file, []byte(*content), 0766); err != nil {
			t.FailNow()
		}
	}
	return file
}

type callChecker struct {
	called atomic.Bool
}

func (c *callChecker) Call() {
	c.called.Store(true)
}

func (c *callChecker) Called() bool {
	return c.called.Load()
}

func (c *callChecker) Reset() {
	c.called.Store(false)
}

func TestDelayedActioner(t *testing.T) {
	t.Run("act with delay", func(t *testing.T) {
		cc := &callChecker{}
		da := NewDelayedActioner(cc.Call, 50*time.Millisecond)
		da.Act()
		require.False(t, cc.Called())
		require.Eventually(t, func() bool {
			return cc.Called()
		}, 100*time.Millisecond, 10*time.Millisecond)

	})
	t.Run("two acts", func(t *testing.T) {
		cc := &callChecker{}
		da := NewDelayedActioner(cc.Call, 50*time.Millisecond)
		da.Act()
		require.False(t, cc.Called())
		require.Eventually(t, func() bool {
			return cc.Called()
		}, 100*time.Millisecond, 10*time.Millisecond)
		cc.Reset()
		da.Act()
		require.False(t, cc.Called())
		require.Eventually(t, func() bool {
			return cc.Called()
		}, 100*time.Millisecond, 10*time.Millisecond)
	})
	t.Run("act again before delay", func(t *testing.T) {
		cc := &callChecker{}
		da := NewDelayedActioner(cc.Call, 50*time.Millisecond)
		da.Act()
		require.False(t, cc.Called())
		require.Never(t, cc.Called, 20*time.Millisecond, 5*time.Millisecond)
		da.Act()
		require.False(t, cc.Called())
		require.Never(t, cc.Called, 20*time.Millisecond, 5*time.Millisecond)
		require.Eventually(t, func() bool {
			return cc.Called()
		}, 100*time.Millisecond, 10*time.Millisecond)
	})
	t.Run("act now", func(t *testing.T) {
		cc := &callChecker{}
		da := NewDelayedActioner(cc.Call, 50*time.Millisecond)
		da.Act()
		require.False(t, cc.Called())
		require.Never(t, cc.Called, 20*time.Millisecond, 5*time.Millisecond)
		da.ActNowIfScheduled()
		require.True(t, cc.Called())
	})
}

func TestConfig(t *testing.T) {
	defaultConfigContent := `{"Conf":"` + os.ExpandEnv("$HOME/.config/yandex-disk/config.cfg") + `","Theme":"dark","Notifications":true,"StartDaemon":true,"StopDaemon":false}`
	logger := SetupLogger(false, os.Stdout)
	t.Run("no file", func(t *testing.T) {
		testFile := makeTempCfgFile(t, nil)
		defer os.Remove(testFile)
		cfg, err := NewConfig(testFile, 50*time.Millisecond, logger)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Eventually(t, func() bool {
			return !NotExists(testFile)
		}, 1000*time.Millisecond, 10*time.Millisecond)
		data, err := os.ReadFile(testFile)
		require.NoError(t, err)
		require.Contains(t, string(data), defaultConfigContent)
	})
	t.Run("empty config file", func(t *testing.T) {
		empty := ""
		testFile := makeTempCfgFile(t, &empty)
		defer os.Remove(testFile)
		cfg, err := NewConfig(testFile, 50*time.Millisecond, logger)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Equal(t, &Config{
			path:          testFile,
			da:            cfg.da,
			log:           logger,
			Conf:          os.ExpandEnv("$HOME/.config/yandex-disk/config.cfg"),
			Theme:         "dark",
			Notifications: true,
			StartDaemon:   true,
			StopDaemon:    false,
		}, cfg)
	})
	t.Run("bad config file", func(t *testing.T) {
		bad := "bad,bad,bad"
		testFile := makeTempCfgFile(t, &bad)
		defer os.Remove(testFile)
		cfg, err := NewConfig(testFile, time.Hour, logger)
		require.Error(t, err)
		require.Nil(t, cfg)
	})
	t.Run("incorrect theme", func(t *testing.T) {
		bad := `{"Theme":"incorrect"}`
		testFile := makeTempCfgFile(t, &bad)
		defer os.Remove(testFile)
		cfg, err := NewConfig(testFile, time.Hour, logger)
		require.Error(t, err)
		require.Nil(t, cfg)
	})
	t.Run("empty JSON", func(t *testing.T) {
		emptyJSON := "{}"
		testFile := makeTempCfgFile(t, &emptyJSON)
		defer os.Remove(testFile)
		cfg, err := NewConfig(testFile, time.Hour, logger)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Equal(t, &Config{
			path:          testFile,
			da:            cfg.da,
			log:           logger,
			Conf:          os.ExpandEnv("$HOME/.config/yandex-disk/config.cfg"),
			Theme:         "dark",
			Notifications: true,
			StartDaemon:   true,
			StopDaemon:    false,
		}, cfg)
	})
	t.Run("correct config", func(t *testing.T) {
		content := `{"Theme":"light","StopDaemon":true,"Notifications":false,"StartDaemon":false,"Conf":"config.cfg"}`
		testFile := makeTempCfgFile(t, &content)
		defer os.Remove(testFile)
		cfg, err := NewConfig(testFile, 50*time.Millisecond, logger)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Equal(t, &Config{
			path:          testFile,
			da:            cfg.da,
			log:           logger,
			Conf:          "config.cfg",
			Theme:         "light",
			Notifications: false,
			StartDaemon:   false,
			StopDaemon:    true,
		}, cfg)
	})
	t.Run("save changed now", func(t *testing.T) {
		content := "{}"
		testFile := makeTempCfgFile(t, &content)
		defer os.Remove(testFile)
		cfg, err := NewConfig(testFile, 50*time.Millisecond, logger)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		os.Remove(testFile)          // remove the file to check that it will be created again on saving
		cfg.SetTheme(cfg.GetTheme()) // change config to mark it as changed and trigger saving after timeout
		require.Never(t, func() bool {
			return !NotExists(testFile)
		}, 20*time.Millisecond, 10*time.Millisecond)
		cfg.SetNotifications(cfg.GetNotifications()) // change another field to reschedule saving
		require.Never(t, func() bool {
			return !NotExists(testFile)
		}, 20*time.Millisecond, 10*time.Millisecond)
		cfg.SetStopDaemon(cfg.GetStopDaemon()) // change another field to reschedule saving
		require.Never(t, func() bool {
			return !NotExists(testFile)
		}, 20*time.Millisecond, 10*time.Millisecond)
		cfg.SetStartDaemon(cfg.GetStartDaemon()) // change another field to reschedule saving
		require.Never(t, func() bool {
			return !NotExists(testFile)
		}, 20*time.Millisecond, 10*time.Millisecond)
		cfg.SaveChangedNow() // save config immediately without waiting for timeout
		require.Eventually(t, func() bool {
			return !NotExists(testFile)
		}, 10*time.Millisecond, 2*time.Millisecond)
		data, err := os.ReadFile(testFile)
		require.NoError(t, err)
		require.Contains(t, string(data), defaultConfigContent)
	})
	t.Run("file can't be read", func(t *testing.T) {
		cfg, err := NewConfig("./", time.Hour, logger)
		require.Error(t, err)
		require.EqualError(t, err, "reading config file error: read ./: is a directory")
		require.Nil(t, cfg)
	})
	t.Run("file path can't be created", func(t *testing.T) {
		cfg, err := NewConfig("/dev/non_existing_device/file", time.Hour, logger)
		require.Error(t, err)
		require.EqualError(t, err, "can't create application configuration path: mkdir /dev/non_existing_device/: permission denied")
		require.Nil(t, cfg)
	})
	t.Run("file can't be written", func(t *testing.T) {
		testFile := "/dev/non_existing_file"
		// catch the output of logger to check that error message is logged
		r, w, err := os.Pipe()
		require.NoError(t, err)
		defer r.Close()
		defer w.Close()
		log := SetupLogger(false, w)
		cfg, err := NewConfig(testFile, time.Hour, log)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		cfg.SaveChangedNow() // try to save config to the file with wrong permissions
		w.Close()            // close the writer to allow reading the output
		out, err := io.ReadAll(r)
		require.NoError(t, err)
		require.Contains(t, string(out), "can't save config file")
		require.Contains(t, string(out), "permission denied")
	})
	t.Run("file in current directory", func(t *testing.T) {
		testFile := "default.cfg"
		defer os.Remove(testFile)
		cfg, err := NewConfig(testFile, 500*time.Millisecond, logger)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		cfg.SaveChangedNow()
		require.Eventually(t, func() bool {
			return !NotExists(testFile)
		}, 10*time.Millisecond, 2*time.Millisecond)
		data, err := os.ReadFile(testFile)
		require.NoError(t, err)
		require.Contains(t, string(data), defaultConfigContent)
	})
}

func readStd(f **os.File) func() string {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	out := *f
	*f = w
	return func() string {
		*f = out
		w.Close()
		b, err := io.ReadAll(r)
		if err != nil {
			panic(err)
		}
		return string(b)
	}
}

func TestGetParams(t *testing.T) {
	tAppName := "testApp"
	tVersion := "test"
	t.Run("wo_params", func(t *testing.T) {
		cfgPath, debug := GetParams(tAppName, []string{tAppName}, tVersion)
		require.Equal(t, os.ExpandEnv("$HOME/.config/"+tAppName+"/default.cfg"), cfgPath)
		require.False(t, debug)
	})
	t.Run("with_debug", func(t *testing.T) {
		cfgPath, debug := GetParams(tAppName, []string{tAppName, "-debug"}, tVersion)
		require.Equal(t, os.ExpandEnv("$HOME/.config/"+tAppName+"/default.cfg"), cfgPath)
		require.True(t, debug)
	})
	t.Run("with_cfg", func(t *testing.T) {
		emptyJSON := "{}"
		cfgFile := makeTempCfgFile(t, &emptyJSON)
		defer os.Remove(cfgFile)
		cfgPath, debug := GetParams(tAppName, []string{tAppName, "-config=" + cfgFile}, tVersion)
		require.Equal(t, cfgFile, cfgPath)
		require.False(t, debug)
	})
	t.Run("with_-h", func(t *testing.T) {
		getOut := readStd(&os.Stderr)
		// help request will call os.Exit(0) that panics the testing
		require.Panics(t, func() { GetParams(tAppName, []string{tAppName, "-h"}, tVersion) })
		out := getOut()
		require.Contains(t, out, "Usage")
	})
	t.Run("with_-version", func(t *testing.T) {
		getOut := readStd(&os.Stdout)
		// version request will call os.Exit(0) that panics the testing
		require.Panics(t, func() { GetParams(tAppName, []string{tAppName, "-version"}, tVersion) })
		out := getOut()
		require.Contains(t, out, tVersion)
	})
}

func TestSetupLogger(t *testing.T) {
	infoMsg := "info_msg"
	debugMsg := "debug_msg"
	t.Run("info", func(t *testing.T) {
		getOut := readStd(&os.Stdout)
		l := SetupLogger(false, os.Stdout)
		l.Debug(debugMsg)
		l.Info(infoMsg)
		out := getOut()
		require.Contains(t, out, infoMsg)
		require.NotContains(t, out, debugMsg)
	})
	t.Run("debug", func(t *testing.T) {
		getOut := readStd(&os.Stdout)
		l := SetupLogger(true, os.Stdout)
		l.Debug(debugMsg)
		l.Info(infoMsg)
		out := getOut()
		require.Contains(t, out, infoMsg)
		require.Contains(t, out, debugMsg)
	})
}

func TestXdgOpen(t *testing.T) {
	resultPath := path.Join(t.TempDir(), "xdg-open-result.txt")
	// mock xdg-open by setting xdgOpenCmd variable to a script that writes its arguments to a file
	script := `#!/bin/sh
echo "$@" > "` + resultPath + `"`
	scriptPath := path.Join(t.TempDir(), "xdg-open-mock.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0766); err != nil {
		t.FailNow()
	}
	xdgOpenCmd = scriptPath
	defer func() {
		xdgOpenCmd = "xdg-open" // restore xdgOpenCmd after test
		os.Remove(scriptPath)
		os.Remove(resultPath)
	}()
	url := "https://example.com"
	err := XdgOpen(url)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return !NotExists(resultPath)
	}, 100*time.Millisecond, 10*time.Millisecond)
	data, err := os.ReadFile(resultPath)
	require.NoError(t, err)
	require.Equal(t, string(data), url+"\n")
}

// 100% test coverage!
