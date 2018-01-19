# yd-go
## Panel indicator for Yandex-disk CLI daemon (linux/GTK+)

It is GTK+ version of indicator. If you are interested in Qt version, visit: https://github.com/slytomcat/yd-qgo

I've made it as it is rather well-known task for me (I've made the similar indicator in YD-tools project in Python language: https://github.com/slytomcat/yandex-disk-indicator it's also uses GTK+).

GUI for wrapper show current status in system tray by different icons. During synchronization the icon is animated. 

Desktop notifications inform user when daemon started/stopped or synchronization started/stopped.

The system try icon has menu that allows:
  - to see the current daemon status and cloud-disk properties (Used/Total/Free/Trash)
  - to see (in submenu) and open (in default program) last synchronized files 
  - to start/stop daemon
  - to see the originl output of daemon in user language
  - to open local syncronized path
  - to open cloud-disk in browser
  - to open help/support page

Application has its configuration file in ~/.config/yd-go/default.cfg file. File is in JSON format and contain following options:
  - "Conf" - path to daemon config file (default "~/.config/yandex-disk/config.cfg"
  - "Theme" - icons theme name (default "dark", may be set to "dark" or "light")
  - "Notifications" - Display or not the desktop notifications (default true)
  - "StartDaemon" - Flag that shows should be the daemon started on app start (default true)
  - "StopDaemon" - Flag that shows should be the daemon stopped on app closure

## Get
Download source from master branch and unzip it to the go source folder ($GOHATH/src) (it can be removed after buiding and installation).
Change current directoru to the progect folder 
    cd $GOHATH/src/yd-go/

## Build 
For building this prject the additional packages are requered:
- packages for GTK & AppIndicator C code compilation: libgtk-3-dev libappindicator3-dev. You can install them (in Debial based Linux distributions):

    `sudo apt-get install libgtk-3-dev libappindicator3-dev`

2. Go packages that is used in the progect. They can be installed by:

    `go get -d`
    
Then you can buld the projec with 

    go build

## Installation
Run install.bash script with root previlegies for installation.

    sudo ./install.bash


## Usage
		yd-go [-debug] [-config=<Path to indicator config>]

	-config string
		Path to the indicator configuration file (default "~/.config/yd.go/default.cfg")
	-debug
		Alow debugging messages to be sent to stderr


Note that yandex-disk CLI utility must be installed and connection to cloud disk mast be configured for usage the yd-go utility.
