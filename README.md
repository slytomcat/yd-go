# yd-go 
[![Go](https://github.com/slytomcat/yd-go/actions/workflows/go.yml/badge.svg?branch=master)](https://github.com/slytomcat/yd-go/actions/workflows/go.yml)
## Panel indicator for Yandex-disk CLI daemon (linux)


This version of indicator uses B-Bus for communication to the status notification plugin. It is fully independent of the desktop environment of Linux distribution.

Russian wiki: https://github.com/slytomcat/yd-go/wiki

I've made it as it is rather well-known task for me: I've made the similar indicator (GTK+ vesion) in YD-tools project in Python language: https://github.com/slytomcat/yandex-disk-indicator.

Indicator shows current status by different icons in the status notification area. During synchronization the icon is animated.

Desktop notifications inform user when daemon started/stopped or synchronization started/stopped.

The status notification icon has menu that allows:
  - to see the current daemon status and cloud-disk properties (Used/Total/Free/Trash)
  - to see (in submenu) and open (in default program) last synchronized files 
  - to start/stop daemon
  - to see the originl output of daemon in user language
  - to open local syncronized path
  - to open cloud-disk in browser
  - to open help/support page

Application uses its configuration file with dafault path ~/.config/yd-go/default.cfg file. File is in JSON format and contain following options:
  - "Conf" - path to daemon config file (default "~/.config/yandex-disk/config.cfg"
  - "Theme" - icons theme name (default "dark", may be set to "dark" or "light")
  - "Notifications" - Display or not the desktop notifications (default true)
  - "StartDaemon" - Flag that shows that the daemon should be started on app start (default true)
  - "StopDaemon" - Flag that shows that the daemon should be stopped on app closure

## Get
Download source from master branch and unzip or just clone repository .

## Build 
There is no additional libraries/packages requered for building. Just jump into project directory and run:

```bash
./build
```
## Installation
Run 
```bash
go install
```

Or copy yd-go to somewhere in the path (/usr/bin for example)

## Usage
		yd-go [-debug] [-config=<Path to indicator config>]

	-config string
		Path to the indicator configuration file (default "~/.config/yd.go/default.cfg")
	-debug
		Alow debugging messages to be sent to stderr


Note that yandex-disk CLI utility must be installed and connection to cloud disk mast be configured for usage the yd-go utility.