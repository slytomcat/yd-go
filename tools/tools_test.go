package tools

import (
	"bytes"
	"io"
	"os"
	"path"
	"testing"

	"github.com/slytomcat/llog"
	"github.com/stretchr/testify/assert"
)

func TestShortName(t *testing.T) {
	assert.Equal(t, "1234567890", MakeTitle("1234567890", 20))
	assert.Equal(t, "12...890", MakeTitle("1234567890", 8))
	assert.Equal(t, "12...123", MakeTitle("1234567890123", 8))
	assert.Equal(t, "русский текст", MakeTitle("русский текст", 20))
	assert.Equal(t, "рус...дада", MakeTitle("русский текст дада", 10))
	su := "one_two"
	sm := MakeTitle(su, 10)
	assert.Equal(t, "one\u2009\u0332\u2009two", sm)
}

func TestNotExists(t *testing.T) {
	wd, err := os.Getwd()
	assert.NoError(t, err)
	assert.False(t, NotExists(wd))
	assert.True(t, NotExists("/Unreal path+*$"))
}

func TestConfig(t *testing.T) {
	testPath := t.TempDir()
	testFile := path.Join(testPath, "cfg")

	t.Run("no config file", func(t *testing.T) {
		cfg := NewConfig(testFile)
		assert.NotNil(t, cfg)
		assert.Equal(t, &Config{
			Conf:          os.ExpandEnv("$HOME/.config/yandex-disk/config.cfg"),
			Theme:         "dark",
			Notifications: true,
			StartDaemon:   true,
			StopDaemon:    false,
		}, cfg)
	})

	t.Run("config file exists", func(t *testing.T) {
		os.WriteFile(testFile, []byte(`{"Conf":"config.cfg","Theme":"none","Notifications":false,"StartDaemon":false,"StopDaemon":true}`), 0766)

		cfg1 := NewConfig(testFile)
		assert.NotNil(t, cfg1)
		assert.Equal(t, &Config{
			Conf:          "config.cfg",
			Theme:         "none",
			Notifications: false,
			StartDaemon:   false,
			StopDaemon:    true,
		}, cfg1)
	})

	t.Run("empty config file", func(t *testing.T) {
		os.WriteFile(testFile, []byte(`{}`), 0766)

		cfg3 := NewConfig(testFile)
		assert.NotNil(t, cfg3)
		assert.Equal(t, &Config{
			Conf:          os.ExpandEnv("$HOME/.config/yandex-disk/config.cfg"),
			Theme:         "dark",
			Notifications: true,
			StartDaemon:   true,
			StopDaemon:    false,
		}, cfg3)
	})

	t.Run("partial config file", func(t *testing.T) {
		os.WriteFile(testFile, []byte(`{"Theme":"none","Notifications":false,"StopDaemon":true}`), 0766)

		cfg3 := NewConfig(testFile)
		assert.NotNil(t, cfg3)
		assert.Equal(t, &Config{
			Conf:          os.ExpandEnv("$HOME/.config/yandex-disk/config.cfg"), // default
			Theme:         "none",                                               // config
			Notifications: false,                                                // config
			StartDaemon:   true,                                                 // default
			StopDaemon:    true,                                                 // config
		}, cfg3)
	})

	t.Run("bad config file", func(t *testing.T) {
		os.WriteFile(testFile, []byte(`bad,bad,bad`), 0766)

		assert.Panics(t, func() { _ = NewConfig(testFile) })
	})

	t.Run("config file cat't be read", func(t *testing.T) {
		assert.Panics(t, func() { _ = NewConfig(testPath) })
	})

	t.Run("config file cat't be written", func(t *testing.T) {
		assert.Panics(t, func() { _ = NewConfig("/dev/non_existing_device") })
	})

	t.Run("config file path cat't be created", func(t *testing.T) {
		assert.Panics(t, func() { _ = NewConfig("/dev/non_existing_device/file") })
	})

	// 100% coverage for Config !!!
}

func TestAppInit(t *testing.T) {
	appName := "yd-go-test"

	t.Run("start w/o params", func(t *testing.T) {
		cfgPath := AppInit(appName, []string{appName})
		assert.Equal(t, os.ExpandEnv("$HOME/.config/yd-go-test/default.cfg"), cfgPath)
	})

	t.Run("start with -config", func(t *testing.T) {
		cfgPath := AppInit(appName, []string{appName, "-config=file"})
		assert.Equal(t, "file", cfgPath)
	})

	t.Run("start with -debug", func(t *testing.T) {

		buf := &bytes.Buffer{}
		llog.SetOutput(buf)
		cfgPath := AppInit(appName, []string{appName, "-debug", "-config=file"})
		assert.Equal(t, "file", cfgPath)
		assert.Contains(t, buf.String(), "Debugging enabled")
	})

	t.Run("start with -h", func(t *testing.T) {
		r, w, err := os.Pipe()
		assert.NoError(t, err)
		oserr := os.Stderr
		os.Stderr = w
		defer func() {
			os.Stderr = oserr
		}()
		// help request will call os.Exit(0) that panics the testing
		assert.Panics(t, func() { _ = AppInit(appName, []string{appName, "-h"}) })
		w.Close()
		b, err := io.ReadAll(r)
		assert.NoError(t, err)
		assert.Contains(t, string(b), "Usage:\n\n\t\t\"yd-go-test")
	})
}

// I have no idea how to test XdgOpen...
