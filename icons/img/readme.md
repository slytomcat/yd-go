Icons are used to display the indicator statuses.

There are two icons themes:
 - `dark*` - for dark panel
 - `light*` - for light panel

Each theme must support the following set of icons:
 - `*Idle.png` - displayed when Yandex.disk is synchronized
 - `*Pause.png` - displayed when Yandex.disk daemon not started or synchronization is paused
 - `*Error.png` - displayed when some error occurs in synchronization
 - `*Busy[1-5].png` - set of icons to indicate the synchronization process.

Icons `*Busy[1-5].png` are displayed sequentially in the loop (to simulate animation):

   `*Busy1.png` -> `*Busy2.png` -> `*Busy3.png` -> `*Busy4.png` -> `*Busy5.png` -> `*Busy1.png` -> `*Busy2.png` ...

The special icon `logo.png` is used into about and into other notifications.
