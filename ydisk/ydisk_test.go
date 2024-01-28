package ydisk

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/slytomcat/llog"
	"github.com/stretchr/testify/require"
)

var (
	Cfg, CfgPath, SyncDir, SymExe string
	YD                            *YDisk
)

const (
	SyncDirPath    = "$HOME/TeSt_Yandex.Disk_TeSt"
	ConfigFilePath = "$HOME/.config/TeSt_Yandex.Disk_TeSt"
)

func TestMain(m *testing.M) {
	flag.Parse()

	// Initialization
	llog.SetLevel(llog.DEBUG)
	llog.SetFlags(log.Lshortfile | log.Lmicroseconds)
	CfgPath = os.ExpandEnv(ConfigFilePath)
	Cfg = filepath.Join(CfgPath, "config.cfg")
	SyncDir = os.ExpandEnv(SyncDirPath)
	os.Setenv("Sim_SyncDir", SyncDir)
	os.Setenv("Sim_ConfDir", CfgPath)
	err := os.MkdirAll(CfgPath, 0755)
	if err != nil {
		log.Fatal(CfgPath, " creation error:", err)
	}

	SymExe, err = exec.LookPath("yandex-disk")
	if err != nil {
		log.Fatal("yandex-disk utility lookup error:", err)
	}

	exec.Command(SymExe, "stop").Run()
	os.RemoveAll(path.Join(os.TempDir(), "yandexdisksimulator.socket"))
	log.Printf("Tests init completed: yd exe: %v", SymExe)

	// Run tests
	e := m.Run()

	// Clearance
	exec.Command(SymExe, "stop").Run()
	os.RemoveAll(path.Join(os.TempDir(), "yandexdisksimulator.socket"))
	os.RemoveAll(CfgPath)
	os.RemoveAll(SyncDir)
	log.Println("Tests clearance completed")
	os.Exit(e)
}

func TestNotInstalled(t *testing.T) {
	t.Setenv("PATH", "")
	// test not_installed case
	yd, err := NewYDisk(Cfg)
	require.Error(t, err)
	require.Nil(t, yd)
}

func TestWrongConf(t *testing.T) {
	// test initialization with wrong/not-existing config
	yd, err := NewYDisk(Cfg + "_bad")
	require.Error(t, err)
	require.Nil(t, yd)
}

func TestEmptyConf(t *testing.T) {
	// test initialization with empty config
	file, err := os.OpenFile(Cfg, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0666)
	require.NoError(t, err)
	defer file.Close()
	_, err = file.Write([]byte("Dir=\"no_dir\"\n\nproxy=\"no\"\n"))
	require.NoError(t, err)
	file.Close()
	defer os.Remove(Cfg)
	_, err = NewYDisk(Cfg)
	require.Error(t, err)
}

func TestFull(t *testing.T) {
	// prepare for similation
	err := exec.Command(SymExe, "setup").Run()
	require.NoError(t, err)
	var YD *YDisk
	var yds YDvals
	t.Run("Create", func(t *testing.T) {
		YD, err = NewYDisk(Cfg)
		require.NoError(t, err)
	})

	t.Run("NotStartedOupput", func(t *testing.T) {
		output := YD.Output()
		require.Empty(t, output)
	})

	t.Run("InitialEvent", func(t *testing.T) {
		require.Eventually(t, func() bool {
			select {
			case yds = <-YD.Changes:
				require.Equal(t, "{none unknown     [] true   }", fmt.Sprintf("%v", yds))
				return true
			default:
				return false
			}
		}, time.Second, 100*time.Millisecond)
	})

	t.Run("Start", func(t *testing.T) {
		err = YD.Start()
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			select {
			case yds = <-YD.Changes:
				require.Equal(t, "{paused none     [File.ods downloads/file.deb downloads/setup download down do_it very_very_long_long_file_with_underscore o w n] true   }", fmt.Sprintf("%v", yds))
				return true
			default:
				return false
			}
		}, time.Second*3, 300*time.Microsecond)
	})

	t.Run("OutputStarted", func(t *testing.T) {
		output := YD.Output()
		require.NotEmpty(t, output)
	})

	t.Run("Start2Idle", func(t *testing.T) {
		require.Eventually(t, func() bool {
			select {
			case yds = <-YD.Changes:
				if yds.Stat != "idle" {
					return false
				}
				require.Equal(t, "{idle index 43.50 GB 2.89 GB 40.61 GB 0 B [File.ods downloads/file.deb downloads/setup download down do_it very_very_long_long_file_with_underscore o w n] false   }", fmt.Sprintf("%v", yds))
				return true
			default:
				return false
			}
		}, 30*time.Second, time.Second)
	})

	t.Run("SecondaryStart", func(t *testing.T) {
		err := YD.Start()
		require.NoError(t, err)
		select {
		case <-YD.Changes:
			t.Error("Event received within 3 sec interval after secondary start of daemon")
		case <-time.After(time.Second * 3):
		}
	})

	t.Run("Sync", func(t *testing.T) {
		err = exec.Command("yandex-disk", "sync").Run()
		require.NoError(t, err)
		select {
		case yds = <-YD.Changes:
			require.Equal(t,
				"{index idle 43.50 GB 2.89 GB 40.61 GB 0 B [File.ods downloads/file.deb downloads/setup download down do_it very_very_long_long_file_with_underscore o w n] false   }",
				fmt.Sprintf("%v", yds))
		case <-time.After(2 * time.Second):
			t.Fatal("no event for 2 seconds after sync command")
		}
	})

	t.Run("Busy2Idle", func(t *testing.T) {
		require.Eventually(t, func() bool {
			select {
			case yds = <-YD.Changes:
				if yds.Stat != "idle" {
					return false
				}
				require.Equal(t,
					"{idle index 43.50 GB 2.89 GB 40.61 GB 0 B [File.ods downloads/file.deb downloads/setup download down do_it very_very_long_long_file_with_underscore o w n] true   }",
					fmt.Sprintf("%v", yds))
				return true
			}
		}, 10*time.Second, time.Second)
	})

	t.Run("Error", func(t *testing.T) {
		require.NoError(t, exec.Command("yandex-disk", "error").Run())
		require.Eventually(t, func() bool {
			select {
			case yds = <-YD.Changes:
				if yds.Stat != "error" {
					return false
				}
				require.Equal(t,
					"{error idle 43.50 GB 2.88 GB 40.62 GB 654.48 MB [File.ods downloads/file.deb downloads/setup download down do_it very_very_long_long_file_with_underscore o w n] false access error downloads/test1 }",
					fmt.Sprintf("%v", yds))
				return true
			default:
				return false
			}

		}, 2*time.Second, 200*time.Millisecond)
	})

	t.Run("Stop", func(t *testing.T) {
		require.NoError(t, YD.Stop())
		require.Eventually(t, func() bool {
			select {
			case yds = <-YD.Changes:
				if yds.Stat != "none" {
					return false
				}
				require.Equal(t, "{none error     [] true   }", fmt.Sprintf("%v", yds))
				return true
			default:
				return false
			}
		}, 3*time.Second, 300*time.Millisecond)
	})

	t.Run("SecondaryStop", func(t *testing.T) {
		require.NoError(t, YD.Stop())
		require.Never(t, func() bool {
			select {
			case <-YD.Changes:
				return true
			default:
				return false
			}
		}, 3*time.Second, 300*time.Millisecond)
	})

	t.Run("Close", func(t *testing.T) {
		YD.Close()
		require.Eventually(t, func() bool {
			select {
			case _, ok := <-YD.Changes:
				require.False(t, ok)
				return true
			default:
				return false
			}
		}, time.Second, 100*time.Millisecond)
	})
}
