# YD.go
## Go experimental wrap for Yandex-disk CLI daemon (linux)

It's my first project on golang.

I've made it as it is rather well-known task for me (I've made the same wrap-up in YD-tools project in Python language).

GUI for wrapper show current status in system tray by different icons. During synchronization the icon is animated. 

Desktop notifications inform user when daemon started/stopped or synchronization started/stopped.

The system try icon has menu that allows:
  - to see the current daemon status and cloud-disk properties (Used/Total/Free/Trash)
  - to start/stop daemon
  - to open local syncronized path
  - to open cloud-disk in browser

Application has its configuration file in ~/.config/yd.go/default.cfg file. File is in JSON format and contain following keys:
  - "Conf" - path to daemon config file (default "~/.config/yandex-disk/config.cfg"
  - "Theme" - icons theme name (default "dark")
  - "Notifications" - Display or not the desktop notifications (default true)
  - "StartDaemon" - Flag that shows should be the daemon started on app start (default true)
  - "StopDaemon" - Flag that shows should be the daemon stopped on app closure
 
 

