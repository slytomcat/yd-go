// Code generated by running "go generate" in golang.org/x/text. DO NOT EDIT.

package main

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
)

type dictionary struct {
	index []uint32
	data  string
}

func (d *dictionary) Lookup(key string) (data string, ok bool) {
	p, ok := messageKeyToIndex[key]
	if !ok {
		return "", false
	}
	start, end := d.index[p], d.index[p+1]
	if start == end {
		return "", false
	}
	return d.data[start:end], true
}

func init() {
	dict := map[string]catalog.Dictionary{
		"en": &dictionary{index: enIndex, data: enData},
		"ru": &dictionary{index: ruIndex, data: ruData},
	}
	fallback := language.MustParse("en")
	cat, err := catalog.NewFromMap(dict, catalog.Fallback(fallback))
	if err != nil {
		panic(err)
	}
	message.DefaultCatalog = cat
}

var messageKeyToIndex = map[string]int{
	"About":                      12,
	"Daemon started":             21,
	"Daemon stopped":             20,
	"Donations":                  13,
	"Free: %s Trash: %s":         19,
	"Help":                       11,
	"Last synchronized":          5,
	"Open YandexDisk in browser": 10,
	"Open: %s":                   9,
	"Quit":                       14,
	"Show daemon output":         8,
	"Start daemon":               6,
	"Status: %s":                 17,
	"Stop daemon":                7,
	"Synchronization finished":   23,
	"Synchronization started":    22,
	"Used: %s/%s":                18,
	"Yandex.Disk daemon output":  15,
	"busy":                       2,
	"idle":                       0,
	"index":                      1,
	"none":                       3,
	"paused":                     4,
	"yd-go is the GTK-based panel indicator for Yandex.Disk daemon.\n\n\tVersion: %s\n\nCopyleft 2017-%s Sly_tom_cat (slytomcat@mail.ru)\n\n\tLicense: GPL v.3\n\n": 16,
}

var enIndex = []uint32{ // 25 elements
	0x00000000, 0x00000005, 0x0000000b, 0x00000010,
	0x00000015, 0x0000001c, 0x0000002e, 0x0000003b,
	0x00000047, 0x0000005a, 0x00000066, 0x00000081,
	0x00000086, 0x0000008c, 0x00000096, 0x0000009b,
	0x000000b5, 0x00000154, 0x00000162, 0x00000174,
	0x0000018d, 0x0000019c, 0x000001ab, 0x000001c3,
	0x000001dc,
} // Size: 124 bytes

const enData string = "" + // Size: 476 bytes
	"\x02idle\x02index\x02busy\x02none\x02paused\x02Last synchronized\x02Star" +
	"t daemon\x02Stop daemon\x02Show daemon output\x02Open: %[1]s\x02Open Yan" +
	"dexDisk in browser\x02Help\x02About\x02Donations\x02Quit\x02Yandex.Disk " +
	"daemon output\x04\x00\x02\x0a\x0a\x98\x01\x02yd-go is the GTK-based pane" +
	"l indicator for Yandex.Disk daemon.\x0a\x0a\x09Version: %[1]s\x0a\x0aCop" +
	"yleft 2017-%[2]s Sly_tom_cat (slytomcat@mail.ru)\x0a\x0a\x09License: GPL" +
	" v.3\x02Status: %[1]s\x02Used: %[1]s/%[2]s\x02Free: %[1]s Trash: %[2]s" +
	"\x02Daemon stopped\x02Daemon started\x02Synchronization started\x02Synch" +
	"ronization finished"

var ruIndex = []uint32{ // 25 elements
	0x00000000, 0x00000011, 0x00000026, 0x00000041,
	0x00000056, 0x00000061, 0x00000099, 0x000000bb,
	0x000000df, 0x0000010a, 0x00000120, 0x0000014e,
	0x0000015b, 0x00000173, 0x0000018e, 0x00000199,
	0x000001bf, 0x00000283, 0x00000297, 0x000002bd,
	0x000002eb, 0x00000311, 0x00000331, 0x00000359,
	0x00000387,
} // Size: 124 bytes

const ruData string = "" + // Size: 903 bytes
	"\x02ожидание\x02индексация\x02синхронизация\x02остановлен\x02пауза\x02По" +
	"следние синхронизированные\x02Запустить утилиту\x02Остановить утилиту" +
	"\x02Показать вывод утилиты\x02Открыть: %[1]s\x02Открыть YandexDisk в бра" +
	"узере\x02Помощь\x02О индикаторе\x02Пожертвования\x02Выход\x02Вывод утил" +
	"иты Yandex.Disk\x04\x00\x02\x0a\x0a\xbd\x01\x02yd-go это индикатор пане" +
	"ли (GTK) для утилиты Yandex.Disk.\x0a\x0a\x09Версия: %[1]s\x0a\x0aCopyl" +
	"eft 2017-%[2]s Sly_tom_cat (slytomcat@mail.ru)\x0a\x0a\x09Лицензия: GPL " +
	"v.3\x02Статус: %[1]s\x02Использовано: %[1]s/%[2]s\x02Свободно: %[1]s Кор" +
	"зина: %[2]s\x02Утилита остановлена\x02Утилита запущена\x02Синхронизация" +
	" начата\x02Синхронизация закончена"

	// Total table size 1627 bytes (1KiB); checksum: D3D26C88
