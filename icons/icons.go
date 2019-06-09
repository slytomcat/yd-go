package icons

import (
	"path"
	"os"
	"fmt"
)
func saveFile(file string, data []byte) error {
	f, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("Can't create file: %v", err)
	}
	defer f.Close()
	
	if _, err = f.Write(data); err != nil {
		return fmt.Errorf("Can't write file: %v", err)
	}
	return nil
}

var (
	// IconBusy - set of 5 icons to be shown in busy status (with animation)
	IconBusy   [5]string 
	// IconError - is the icon to show error status
	IconError  string
	// IconIdle - is shown whe daemon do nothing (waits fo events)
	IconIdle   string
	// IconPause - is shown in inactive status (not started/paused)
	IconPause  string
	// IconNotify - 128x128 icon to show in notifications
	IconNotify string

	icoHome string   // temporary directory for icons files
	)

// PrepareIcons prepare icons files for indicator
func PrepareIcons() error {
	// Get temporary dir icoHome
	icoHome = "/tmp/yd-go-icons"
	if err := os.MkdirAll(icoHome, 0766); err != nil {
		return fmt.Errorf("Can't create icon folder path: %v", err)
	}

	// put all binary data to files
	if err := saveFile(path.Join(icoHome, "darkBusy1.png"), darkBusy1); err != nil {
		return fmt.Errorf("Can't write icon file: %v", err)
		}
	if err := saveFile(path.Join(icoHome, "darkBusy2.png"), darkBusy2); err != nil {
		return fmt.Errorf("Can't write icon file: %v", err)
		}
	if err := saveFile(path.Join(icoHome, "darkBusy3.png"), darkBusy3); err != nil {
		return fmt.Errorf("Can't write icon file: %v", err)
		}
	if err := saveFile(path.Join(icoHome, "darkBusy4.png"), darkBusy4); err != nil {
		return fmt.Errorf("Can't write icon file: %v", err)
		}
	if err := saveFile(path.Join(icoHome, "darkBusy5.png"), darkBusy5); err != nil {
		return fmt.Errorf("Can't write icon file: %v", err)
		}
	if err := saveFile(path.Join(icoHome, "darkError.png"), darkError); err != nil {
		return fmt.Errorf("Can't write icon file: %v", err)
		}
	if err := saveFile(path.Join(icoHome, "darkIdle.png"), darkIdle); err != nil {
		return fmt.Errorf("Can't write icon file: %v", err)
		}
	if err := saveFile(path.Join(icoHome, "darkPause.png"), darkPause); err != nil {
		return fmt.Errorf("Can't write icon file: %v", err)
		}
	if err := saveFile(path.Join(icoHome, "lightBusy1.png"), lightBusy1); err != nil {
		return fmt.Errorf("Can't write icon file: %v", err)
		}
	if err := saveFile(path.Join(icoHome, "lightBusy2.png"), lightBusy2); err != nil {
		return fmt.Errorf("Can't write icon file: %v", err)
		}
	if err := saveFile(path.Join(icoHome, "lightBusy3.png"), lightBusy3); err != nil {
		return fmt.Errorf("Can't write icon file: %v", err)
		}
	if err := saveFile(path.Join(icoHome, "lightBusy4.png"), lightBusy4); err != nil {
		return fmt.Errorf("Can't write icon file: %v", err)
		}
	if err := saveFile(path.Join(icoHome, "lightBusy5.png"), lightBusy5); err != nil {
		return fmt.Errorf("Can't write icon file: %v", err)
		}
	if err := saveFile(path.Join(icoHome, "lightError.png"), lightError); err != nil {
		return fmt.Errorf("Can't write icon file: %v", err)
		}
	if err := saveFile(path.Join(icoHome, "lightIdle.png"), lightIdle); err != nil {
		return fmt.Errorf("Can't write icon file: %v", err)
		}
	if err := saveFile(path.Join(icoHome, "lightPause.png"), lightPause); err != nil {
		return fmt.Errorf("Can't write icon file: %v", err)
		}
	IconNotify = path.Join(icoHome, "yd128.png")
	if err := saveFile(IconNotify, yd128); err != nil {
		return fmt.Errorf("Can't write icon file: %v", err)
		}

	return nil
}
// ClearIcons removes icons form file system on exit
func ClearIcons() error { 
	if err := os.RemoveAll(icoHome); err != nil {
		return fmt.Errorf("Can't remove icon folder: %v", err)
	}
	return nil
}

// SetTheme sets the Icon* variable according to selected theme
func SetTheme(theme string) {

	IconBusy = [5]string{
		path.Join(icoHome, theme + "Busy1.png"),
		path.Join(icoHome, theme + "Busy2.png"),
		path.Join(icoHome, theme + "Busy3.png"),
		path.Join(icoHome, theme + "Busy4.png"),
		path.Join(icoHome, theme + "Busy5.png"),
	}
	IconError = path.Join(icoHome, theme + "Error.png")
	IconIdle = path.Join(icoHome, theme + "Idle.png")
	IconPause = path.Join(icoHome, theme + "Pause.png")
}
