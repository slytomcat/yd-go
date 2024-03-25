# yd-go 
[![Go](https://github.com/slytomcat/yd-go/actions/workflows/go.yml/badge.svg?branch=master)](https://github.com/slytomcat/yd-go/actions/workflows/go.yml)
## Panel indicator for Yandex-disk CLI daemon (linux)

[![Screenshot](https://github.com/slytomcat/yd-go/blob/master/Screenshots/indicator%2Bmenu.png)](https://github.com/slytomcat/yd-go/blob/master/Screenshots/indicator%2Bmenu.png)

This version of indicator uses B-Bus for communication to the status notification plugin. Therefore it's fully independent of the desktop environment of Linux distribution.

IMPORTANT: 

Indicator responsible only for showing the synchronization status in the desktop panel. All the synchronization operations are performed by [yandex-disk utility from Yandex](https://yandex.ru/support/disk-desktop-linux/index.html).

WIKI:

Russian wiki: https://github.com/slytomcat/yd-go/wiki

STORY:

I've made it as it is rather well-known task for me: I've made the similar indicator (GTK+ version) in YD-tools project in Python language: https://github.com/slytomcat/yandex-disk-indicator.

DESCRIPTION:

Indicator shows current status by different icons in the status notification area. During synchronization the icon is animated. Indicator supports dark and light themes. The current theme can be changed via menu.   

Desktop notifications inform user when daemon started/stopped or synchronization started/stopped. Notifications can be switched off.

The status notification icon has menu that allows to:
  - see the current daemon status and cloud-disk properties (Used/Total/Free/Trash)
  - see paths of the last synchronized files and open them (in default program) 
  - start/stop daemon
  - see the original output of daemon in the current user language
  - open local synchronized path
  - open Yandex.Disk in browser
  - open help/support page
  - change the indicator settings (see "Theme", "Notifications", "StartDaemon" and "StopDaemon" settings below)


Application uses its configuration file with default path ~/.config/yd-go/default.cfg file. File is in JSON format and contain following options:
  - "Conf" - Path to daemon config file (default "~/.config/yandex-disk/config.cfg"). This setting can be changed by the -config application option. 
  - "Theme" - Icons theme name (default "dark", may be set to "dark" or "light"). This setting can be changed via indicator menu.
  - "Notifications" - Display or not the desktop notifications (default true). This setting can be changed via indicator menu.
  - "StartDaemon" - Flag that shows that the daemon should be started on app start (default true). This setting can be changed via indicator menu.
  - "StopDaemon" - Flag that shows that the daemon should be stopped on app closure. This setting can be changed via indicator menu.

## Get
Download linux-amd64 binary from [releases](https://github.com/slytomcat/yd-go/releases), copy it to path in PATH (/usr/local/bin for example) and make it executable. 

OR 

Get source from master branch and unzip it or just clone repository build it and install as described below.

## Build 
You must have Golang v1.20+ installed to build the binary. There is no additional libraries/packages required for building. Just jump into project directory and run:

```bash
./build.sh
```
## Installation
Run 
```bash
sudo cp yd-go /usr/local/bin/
```

## Usage
		yd-go [-debug] [-config=<Path to indicator config>]

	-config string
		Path to the indicator configuration file (default "~/.config/yd.go/default.cfg")
	-debug
		Alow debugging messages to be sent to stderr
	-version
		Print out version information and exit


NOTE: the yandex-disk CLI utility must be installed and configured before starting of the yd-go.
