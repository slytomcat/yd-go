# yd-go
[![Go](https://github.com/slytomcat/yd-go/actions/workflows/go.yml/badge.svg?branch=master)](https://github.com/slytomcat/yd-go/actions/workflows/go.yml)

This is a panel indicator for Yandex-disk CLI daemon for linux platforms.

[![Screenshot](https://github.com/slytomcat/yd-go/blob/master/Screenshots/indicator%2Bmenu.png)](https://github.com/slytomcat/yd-go/blob/master/Screenshots/indicator%2Bmenu.png)

## Description

This indicator uses D-Bus for communication to the status notification plugin. Therefore it's fully independent of the desktop environment of Linux distribution. But requires the status notification service implementation for the deskktop environment.

### IMPORTANT

Indicator responsible only for showing the synchronization status in the desktop panel. All the synchronization operations are performed by [yandex-disk utility (CLI) from Yandex](https://yandex.ru/support/disk-desktop-linux/index.html).

Before strating of the indicator application the yandex-disk CLI utility must be installed and properly configured.

### WIKI

Russian wiki: is [here](https://github.com/slytomcat/yd-go/wiki).

### The projesct story

I've made this application as it was rather well-known task for me: I've made the similar indicator (GTK+ version) into [YD-tools project (Python)](https://github.com/slytomcat/yandex-disk-indicator) before. And when I started to learn the golang the rewriting of the already implemented solution were rather obvious task to practice a new programming language. Initially there was two versions of indicators in golang: for GTK+ and for QT. But later I've adopted new version of indicator library that uses D-Bus for the implementation of user interface.

### Indicator

The indicator shows current synchronization status by different icons in the status notification area. During synchronization the icon is animated to show that synchronization is in process. Indicator supports dark and light descktop themes. The current theme can be changed into menu.

Desktop notifications (popup messages) inform user when daemon started/stopped or synchronization started/stopped. Notifications can be switched on or off into menu.

The notification icon has a menu that allows to:
  - see the current daemon status and cloud-disk properties (Used/Total/Free/Trash sizes)
  - see paths of the last synchronized files and open them (into default application for their types)
  - start or stop the synchronisation utilty (yandex-disk CLI utility from yandex)
  - see the original output of `yandex-disk staus` command in the current user language
  - open local synchronized path into the default file-manager
  - open Yandex.Disk in the default Internet browser
  - open help/support web-page
  - change the indicator settings (see `"Theme"`, `"Notifications"`, `"StartDaemon"` and `"StopDaemon"` settings below)


The indicator application uses settings from the configuration file. The default path to configuration file is `~/.config/yd-go/default.cfg`. The path can be changed by the `-config` application commandline start option. The configuration file is in JSON format and it contain following options:
  - `"Conf"` - Path to daemon config file (default: `"~/.config/yandex-disk/config.cfg"`).
  - `"Theme"` - Icons theme name (default: `"dark"`, may be set only to `"dark"` or `"light"`). This setting can be changed into the indicator menu.
  - `"Notifications"` - Display or not the desktop notifications (default: `true`). This setting can be changed into the indicator menu.
  - `"StartDaemon"` - Flag that makes the daemon started on application start (default: `true`). This setting can be changed into indicator menu.
  - `"StopDaemon"` - Flag that cause stop the daemon on application closure (default: `false`). This setting can be changed into indicator menu.

## Installation
### Using prebuild binary

Download linux-amd64 binary from [releases](https://github.com/slytomcat/yd-go/releases), make it executable and copy it to directory that is in the PATH (/usr/local/bin for example).

Example:

     curl -sL  https://github.com/slytomcat/yd-go/releases/latest/download/yd-go > yd-go

or

     wget https://github.com/slytomcat/yd-go/releases/latest/download/yd-go

then

     chmod a+x yd-go
	 sudo mv yd-go /usr/loacl/bin/

After that You can run it as `yd-go` or add it to one of auto-run facilities available itho your DE/OS.

### Build yd-go from sources

#### Prerequirements
You need to have only `git` and `golang v1.24+` to be installed into your OS for building the indicator application.

#### Steps

1. Clone repo: `git clone https://github.com/slytomcat/yd-go.git`
2. Enter the repo root: `cd yd-go`
3. Build the application: `./build.sh` 
4. Copy it to directory that is mentioned into the PATH (for example: /usr/loacl/bin): `sudo cp yd-go /usr/local/bin/`
5. Use it: `yd-go` or add it into one of auto-run facilities available into your DE/OS.

__NOTE__
When `upx` utility is available then the binary will be additionally compressed. If `upx` is not installed into your OS then the binary will be uncompressed and a warning appears abut it. You can use both compressed and not compressed binary, the only difference is the used space on disk for binary (not soo much in both cases).

## The application usage

		yd-go [-debug] [-config=<Path to indicator config>]

	-config string
		Path to the indicator configuration file (default "~/.config/yd.go/default.cfg")
	-debug
		Alow debugging messages to be sent to stderr
	-version
		Print out version information and exit

## Icons

All the indicator icons are embedded into binary during the build time. But You can change them and rebuild the indicator from source. See more details about icons into [icons/img/readme.md](icons/img/readme.md).
