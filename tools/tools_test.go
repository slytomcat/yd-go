package tools

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShortName(t *testing.T) {
	assert.Equal(t, "1234567890", ShortName("1234567890", 20))
	assert.Equal(t, "12...890", ShortName("1234567890", 8))
	assert.Equal(t, "12...123", ShortName("1234567890123", 8))
	assert.Equal(t, "русский текст", ShortName("русский текст", 20))
	assert.Equal(t, "рус...дада", ShortName("русский текст дада", 10))
}

func TestNotExists(t *testing.T) {
	wd, err := os.Getwd()
	assert.NoError(t, err)
	assert.False(t, NotExists(wd))
	assert.True(t, NotExists("/Unreal path+*$"))
}

func TestConfig(t *testing.T) {
	testPath := "./cfgTest/"
	testFile := path.Join(testPath, "cfg")
	defer os.RemoveAll(testPath)
	os.RemoveAll(testPath)

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

		defer func() {
			assert.NotNil(t, recover())
		}()
		_ = NewConfig(testFile)

		assert.Fail(t, "this code should not executed")
	})

	t.Run("config file cat't be read", func(t *testing.T) {

		defer func() {
			assert.NotNil(t, recover())
		}()
		_ = NewConfig(testPath)

		assert.Fail(t, "this code should not executed")
	})

	t.Run("config file cat't be written", func(t *testing.T) {

		defer func() {
			assert.NotNil(t, recover())
		}()
		_ = NewConfig("/dev/non_existing_device")

		assert.Fail(t, "this code should not executed")
	})

	t.Run("config file path cat't be created", func(t *testing.T) {

		defer func() {
			assert.NotNil(t, recover())
		}()
		_ = NewConfig("/dev/non_existing_device/file")

		assert.Fail(t, "this code should not executed")
	})

	// 100% coverage for Config !!!
}

// I have no idea how to test XdgOpen...

// Need tests for AppInit
