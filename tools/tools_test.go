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

func makeTempFile(t *testing.T, content string) string {
	file := path.Join(t.TempDir(), "cfg")
	if err := os.WriteFile(file, []byte(content), 0766); err != nil {
		panic(err)
	}
	return file
}

func TestConfig(t *testing.T) {
	t.Run("no config file", func(t *testing.T) {
		testFile := makeTempFile(t, "")
		os.Remove(testFile)
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
		testFile := makeTempFile(t, `{"Conf":"config.cfg","Theme":"dark","Notifications":false,"StartDaemon":false,"StopDaemon":true}`)
		defer os.Remove(testFile)
		cfg, err := NewConfig(testFile)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Equal(t, &Config{
			path:          testFile,
			Conf:          "config.cfg",
			Theme:         "dark",
			Notifications: false,
			StartDaemon:   false,
			StopDaemon:    true,
		}, cfg)
	})

	t.Run("empty config file", func(t *testing.T) {
		testFile := makeTempFile(t, `{}`)
		defer os.Remove(testFile)
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

	t.Run("partial config file", func(t *testing.T) {
		testFile := makeTempFile(t, `{"Theme":"dark","Notifications":false,"StopDaemon":true}`)
		defer os.Remove(testFile)
		cfg, err := NewConfig(testFile)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Equal(t, &Config{
			path:          testFile,
			Conf:          os.ExpandEnv("$HOME/.config/yandex-disk/config.cfg"), // default
			Theme:         "dark",                                               // config
			Notifications: false,                                                // config
			StartDaemon:   true,                                                 // default
			StopDaemon:    true,                                                 // config
		}, cfg)
	})

	t.Run("incorrect theme", func(t *testing.T) {
		testFile := makeTempFile(t, `{"Theme":"incorrect"}`)
		defer os.Remove(testFile)
		cfg, err := NewConfig(testFile)
		require.Error(t, err)
		require.Nil(t, cfg)
	})

	t.Run("bad config file", func(t *testing.T) {
		testFile := makeTempFile(t, `bad,bad,bad`)
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

func TestAppInit(t *testing.T) {
	appName := "yd-go-test"

	t.Run("start w/o params", func(t *testing.T) {
		cfg, msg, log := AppInit(appName, []string{appName}, "test")
		require.NotNil(t, cfg)
		require.NotNil(t, msg)
		require.NotNil(t, log)
		require.Equal(t, os.ExpandEnv("$HOME/.config/yd-go-test/default.cfg"), cfg.path)
	})

	t.Run("start with -config", func(t *testing.T) {
		cfgFile := makeTempFile(t, "{}")
		defer os.Remove(cfgFile)
		cfg, msg, log := AppInit(appName, []string{appName, "-config=" + cfgFile}, "test")
		require.NotNil(t, cfg)
		require.NotNil(t, msg)
		require.NotNil(t, log)
		require.Equal(t, cfgFile, cfg.path)
	})

	t.Run("start with -debug", func(t *testing.T) {
		cfgFile := makeTempFile(t, "{}")
		defer os.Remove(cfgFile)
		cfg, msg, log := AppInit(appName, []string{appName, "-debug", "-config=" + cfgFile}, "test")
		require.NotNil(t, cfg)
		require.NotNil(t, msg)
		require.NotNil(t, log)
		require.Equal(t, cfgFile, cfg.path)
	})

	t.Run("start with -h", func(t *testing.T) {
		r, w, err := os.Pipe()
		require.NoError(t, err)
		osErr := os.Stderr
		os.Stderr = w
		defer func() {
			os.Stderr = osErr
		}()
		// help request will call os.Exit(0) that panics the testing
		require.Panics(t, func() { _, _, _ = AppInit(appName, []string{appName, "--help"}, "test") })
		w.Close()
		b, err := io.ReadAll(r)
		require.NoError(t, err)
		require.Contains(t, string(b), "Usage:\n\n\t\t\"yd-go-test")
	})
	t.Run("config reading error", func(t *testing.T) {
		r, w, err := os.Pipe()
		require.NoError(t, err)
		osErr := os.Stdout
		os.Stdout = w
		defer func() {
			os.Stdout = osErr
		}()
		require.NotPanics(t, func() { _, _, _ = AppInit(appName, []string{appName, "--config=/dev/"}, "test") })
		w.Close()
		b, err := io.ReadAll(r)
		require.NoError(t, err)
		require.Contains(t, string(b), "level=ERROR msg=\"getting config error\" error=\"reading config file error: read /dev/: is a directory\"")
	})

	t.Run("start with version", func(t *testing.T) {
		r, w, err := os.Pipe()
		require.NoError(t, err)
		osStdOut := os.Stdout
		os.Stdout = w
		defer func() {
			os.Stdout = osStdOut
		}()
		// help request will call os.Exit(0) that panics the testing
		require.Panics(t, func() { _, _, _ = AppInit(appName, []string{appName, "--version"}, "test") })
		w.Close()
		b, err := io.ReadAll(r)
		require.NoError(t, err)
		require.Contains(t, string(b), "ver.:")
	})
}

// I have no idea how to test XdgOpen...
