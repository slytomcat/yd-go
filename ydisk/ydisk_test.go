package ydisk

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/slytomcat/llog"
)

func init() {
	llog.SetLevel(llog.DEBUG)
}

func TestFailInit(t *testing.T) {
	home := os.Getenv("HOME")
	cfg := filepath.Join(home, ".config", "yandex-disk", "config.cfg")
	file, err := os.Open(cfg)
	if err != nil {
		llog.Critical(err)
	}
	
	origCfg := make([]byte, 1024)
	n, err := file.Read(origCfg)
	if err != nil {
		llog.Critical(err)
	}
	if n ==0 {
		llog.Critical("Empty daemon configuration")
	}
	file.Close()

	file, err = os.Open(cfg)
	if err != nil {
		llog.Critical(err)
	}
	n, err = file.Write([]byte("proxy=\"no\""))
	file.Close()
	defer func(){
		fo, err := os.Open(cfg)
		if err != nil {
			llog.Critical(err)
		}
		n, err = fo.Write(origCfg)
		if n ==0 || err != nil {
			llog.Critical("Can't restore daemon configuration")
		}
	}()

	defer func(){
		_ = recover()
	}()
	_ = NewYDisk(cfg)

	t.Error("Initialized with empty daemon config file")
}