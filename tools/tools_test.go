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

func TestConfig(t *testing.T) {
	testPath := t.TempDir()
	testFile := path.Join(testPath, "cfg")
	defer os.Remove(testFile)

	t.Run("no config file", func(t *testing.T) {
		cfg, err := NewConfig(testFile)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Equal(t, &Config{
			path:          testFile,
			Conf:          os.ExpandEnv("$HOME/.config/yandex-disk/config.cfg"),
			Theme:         "dark",
			Notifications: true,
			StartDaemon:   true,
			StopDaemon:    false,
		}, cfg)
	})

	t.Run("config file exists", func(t *testing.T) {
		err := os.WriteFile(testFile, []byte(`{"Conf":"config.cfg","Theme":"none","Notifications":false,"StartDaemon":false,"StopDaemon":true}`), 0766)
		require.NoError(t, err)
		cfg1, err := NewConfig(testFile)
		require.NoError(t, err)
		require.NotNil(t, cfg1)
		require.Equal(t, &Config{
			path:          testFile,
			Conf:          "config.cfg",
			Theme:         "none",
			Notifications: false,
			StartDaemon:   false,
			StopDaemon:    true,
		}, cfg1)
	})

	t.Run("empty config file", func(t *testing.T) {
		err := os.WriteFile(testFile, []byte(`{}`), 0766)
		require.NoError(t, err)
		cfg3, err := NewConfig(testFile)
		require.NoError(t, err)
		require.NotNil(t, cfg3)
		require.Equal(t, &Config{
			path:          testFile,
			Conf:          os.ExpandEnv("$HOME/.config/yandex-disk/config.cfg"),
			Theme:         "dark",
			Notifications: true,
			StartDaemon:   true,
			StopDaemon:    false,
		}, cfg3)
	})

	t.Run("partial config file", func(t *testing.T) {
		err := os.WriteFile(testFile, []byte(`{"Theme":"none","Notifications":false,"StopDaemon":true}`), 0766)
		require.NoError(t, err)
		cfg3, err := NewConfig(testFile)
		require.NoError(t, err)
		require.NotNil(t, cfg3)
		require.Equal(t, &Config{
			path:          testFile,
			Conf:          os.ExpandEnv("$HOME/.config/yandex-disk/config.cfg"), // default
			Theme:         "none",                                               // config
			Notifications: false,                                                // config
			StartDaemon:   true,                                                 // default
			StopDaemon:    true,                                                 // config
		}, cfg3)
	})

	t.Run("bad config file", func(t *testing.T) {
		err := os.WriteFile(testFile, []byte(`bad,bad,bad`), 0766)
		require.NoError(t, err)
		cfg, err := NewConfig(testFile)
		require.Error(t, err)
		require.Nil(t, cfg)
	})

	t.Run("config file cat't be read", func(t *testing.T) {
		cfg, err := NewConfig(testPath)
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

func TestAppInit(t *testing.T) {
	appName := "yd-go-test"

	t.Run("start w/o params", func(t *testing.T) {
		cfgPath, debug := AppInit(appName, []string{appName}, "test")
		require.Equal(t, os.ExpandEnv("$HOME/.config/yd-go-test/default.cfg"), cfgPath)
		require.False(t, debug)
	})

	t.Run("start with -config", func(t *testing.T) {
		cfgPath, debug := AppInit(appName, []string{appName, "-config=file"}, "test")
		require.Equal(t, "file", cfgPath)
		require.False(t, debug)
	})

	t.Run("start with -debug", func(t *testing.T) {
		cfgPath, debug := AppInit(appName, []string{appName, "-debug", "-config=file"}, "test")
		require.Equal(t, "file", cfgPath)
		require.True(t, debug)
	})

	t.Run("start with -h", func(t *testing.T) {
		r, w, err := os.Pipe()
		require.NoError(t, err)
		oserr := os.Stderr
		os.Stderr = w
		defer func() {
			os.Stderr = oserr
		}()
		// help request will call os.Exit(0) that panics the testing
		require.Panics(t, func() { _, _ = AppInit(appName, []string{appName, "-h"}, "test") })
		w.Close()
		b, err := io.ReadAll(r)
		require.NoError(t, err)
		require.Contains(t, string(b), "Usage:\n\n\t\t\"yd-go-test")
	})
	// t.Run("start with version", func(t *testing.T) {
	// 	r, w, err := os.Pipe()
	// 	require.NoError(t, err)
	// 	oserr := os.Stderr
	// 	os.Stderr = w
	// 	defer func() {
	// 		os.Stderr = oserr
	// 	}()
	// 	// help request will call os.Exit(0) that panics the testing
	// 	require.Panics(t, func() { _, _ = AppInit(appName, []string{appName, "version"}, "test") })
	// 	w.Close()
	// 	b, err := io.ReadAll(r)
	// 	require.NoError(t, err)
	// 	require.Contains(t, string(b), "Usage:\n\n\t\t\"yd-go-test")
	// })
}

// I have no idea how to test XdgOpen...
