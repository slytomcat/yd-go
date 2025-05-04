package tools

import (
	"io"
	"os"
	"path"
	"testing"

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

func makeTempCfgFile(t *testing.T, content string) string {
	file := path.Join(t.TempDir(), "default.cfg")
	if err := os.WriteFile(file, []byte(content), 0766); err != nil {
		panic(err)
	}
	return file
}

func TestConfig(t *testing.T) {
	t.Run("no config file", func(t *testing.T) {
		testFile := makeTempCfgFile(t, "")
		os.Remove(testFile)
		cfg, err := NewConfig(testFile)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Equal(t, &Config{
			path:          testFile,
			ID:            "default.cfg",
			Conf:          os.ExpandEnv("$HOME/.config/yandex-disk/config.cfg"),
			Theme:         "dark",
			Notifications: true,
			StartDaemon:   true,
			StopDaemon:    false,
		}, cfg)
	})

	t.Run("config file exists", func(t *testing.T) {
		testFile := makeTempCfgFile(t, `{"Conf":"config.cfg","Theme":"dark","Notifications":false,"StartDaemon":false,"StopDaemon":true}`)
		defer os.Remove(testFile)
		cfg, err := NewConfig(testFile)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Equal(t, &Config{
			path:          testFile,
			ID:            "default.cfg",
			Conf:          "config.cfg",
			Theme:         "dark",
			Notifications: false,
			StartDaemon:   false,
			StopDaemon:    true,
		}, cfg)
	})

	t.Run("empty config file", func(t *testing.T) {
		testFile := makeTempCfgFile(t, `{}`)
		defer os.Remove(testFile)
		cfg, err := NewConfig(testFile)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Equal(t, &Config{
			path:          testFile,
			ID:            "default.cfg",
			Conf:          os.ExpandEnv("$HOME/.config/yandex-disk/config.cfg"),
			Theme:         "dark",
			Notifications: true,
			StartDaemon:   true,
			StopDaemon:    false,
		}, cfg)
	})

	t.Run("partial config file", func(t *testing.T) {
		testFile := makeTempCfgFile(t, `{"Theme":"dark","Notifications":false,"StopDaemon":true}`)
		defer os.Remove(testFile)
		cfg, err := NewConfig(testFile)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Equal(t, &Config{
			path:          testFile,
			ID:            "default.cfg",
			Conf:          os.ExpandEnv("$HOME/.config/yandex-disk/config.cfg"), // default
			Theme:         "dark",                                               // config
			Notifications: false,                                                // config
			StartDaemon:   true,                                                 // default
			StopDaemon:    true,                                                 // config
		}, cfg)
	})

	t.Run("incorrect theme", func(t *testing.T) {
		testFile := makeTempCfgFile(t, `{"Theme":"incorrect"}`)
		defer os.Remove(testFile)
		cfg, err := NewConfig(testFile)
		require.Error(t, err)
		require.Nil(t, cfg)
	})

	t.Run("bad config file", func(t *testing.T) {
		testFile := makeTempCfgFile(t, `bad,bad,bad`)
		defer os.Remove(testFile)
		cfg, err := NewConfig(testFile)
		require.Error(t, err)
		require.Nil(t, cfg)
	})

	t.Run("config file cat't be read", func(t *testing.T) {
		cfg, err := NewConfig("/dev/")
		require.Error(t, err)
		require.Nil(t, cfg)
	})

	t.Run("config file cat't be written", func(t *testing.T) {
		cfg, err := NewConfig("/dev/non_existing_device")
		require.Error(t, err)
		require.Nil(t, cfg)
	})

	t.Run("config file path cat't be created", func(t *testing.T) {
		cfg, err := NewConfig("/dev/non_existing_device/file")
		require.Error(t, err)
		require.Nil(t, cfg)
	})

	// 100% coverage for Config !!!
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
		require.Equal(t, "$HOME/.config/"+tAppName+"/default.cfg", cfgPath)
		require.False(t, debug)
	})
	t.Run("with_debug", func(t *testing.T) {
		cfgPath, debug := GetParams(tAppName, []string{tAppName, "-debug"}, tVersion)
		require.Equal(t, "$HOME/.config/"+tAppName+"/default.cfg", cfgPath)
		require.True(t, debug)
	})
	t.Run("with_cfg", func(t *testing.T) {
		cfgFile := makeTempCfgFile(t, "{}")
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
		l := SetupLogger(false)
		l.Debug(debugMsg)
		l.Info(infoMsg)
		out := getOut()
		require.Contains(t, out, infoMsg)
		require.NotContains(t, out, debugMsg)
	})
	t.Run("debug", func(t *testing.T) {
		getOut := readStd(&os.Stdout)
		l := SetupLogger(true)
		l.Debug(debugMsg)
		l.Info(infoMsg)
		out := getOut()
		require.Contains(t, out, infoMsg)
		require.Contains(t, out, debugMsg)
	})
}

// I have no idea how to test XdgOpen...
