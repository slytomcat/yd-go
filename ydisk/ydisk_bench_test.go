package ydisk

import (
	"bytes"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	rPar  = regexp.MustCompile(`\s*(.*): '?(.*?)'?\n`)
	rList = regexp.MustCompile(`: '(.*)'\n`)
	st1   = "Sync progress: 139.38 MB/ 139.38 MB (100 %)\nSynchronization core status: index\nPath to Yandex.Disk directory: '/home/stc/Yandex.Disk'\n\tTotal: 43.50 GB\n\tUsed: 2.89 GB\n\tAvailable: 40.61 GB\n\tMax file size: 50 GB\n\tTrash size: 0 B\n\nLast synchronized items:\n\tfile: 'NewFile'\n\tfile: 'File.ods'\n\tfile: 'downloads/file.deb'\n\tfile: 'downloads/setup'\n\tfile: 'download'\n\tfile: 'down'\n\tfile: 'do'\n\tfile: 'd'\n\tfile: 'o'\n\tfile: 'w'\n\n"
	st2   = "Synchronization core status: idle\nPath to Yandex.Disk directory: '/home/stc/Yandex.Disk'\n\tTotal: 43.50 GB\n\tUsed: 2.89 GB\n\tAvailable: 40.61 GB\n\tMax file size: 50 GB\n\tTrash size: 0 B\n\nLast synchronized items:\n\tfile: 'File.ods'\n\tfile: 'downloads/file.deb'\n\tfile: 'downloads/setup'\n\tfile: 'download'\n\tfile: 'down'\n\tfile: 'do'\n\tfile: 'd'\n\tfile: 'o'\n\tfile: 'w'\n\tfile: 'n'\n\n"
)

func BenchmarkYDvalUpdateString(b *testing.B) {
	yd := newYDvals()
	for b.Loop() {
		yd.update(st1)
		yd.update(st2)
	}
}
func BenchmarkYDvalUpdatePreComp(b *testing.B) {
	yd := newYDvals()
	for b.Loop() {
		yd.update1(st1)
		yd.update1(st2)
	}
}

func BenchmarkYDvalUpdateOrig(b *testing.B) {
	yd := newYDvals()
	for b.Loop() {
		yd.update2(st1)
		yd.update2(st2)
	}
}

// func BenchmarkYDiskGetOutput(b *testing.B) {
// 	// prepare for simulation
// 	err := exec.Command(SymExe, "setup").Run()
// 	if err != nil {
// 		b.Fatal("simulation prepare error")
// 	}
// 	out, err := exec.Command(SymExe, "start").Output()
// 	if err != nil {
// 		b.Fatal("simulation prepare error " + SymExe + err.Error() + string(out))
// 	}
// 	<-time.After(time.Second)
// 	defer func() {
// 		err := exec.Command(SymExe, "stop").Run()
// 		if err != nil {
// 			b.Fatal("simulation prepare error " + SymExe + err.Error() + string(out))
// 		}
// 	}()

// 	for range b.N {
// 		st, err := exec.Command(SymExe, "status").Output()
// 		if err != nil {
// 			b.Fatal("simulation prepare error " + SymExe + err.Error())
// 		}
// 		if len(st) == 0 {
// 			b.Fatal("simulation error: empty output")
// 		}

// 	}
// }

// func BenchmarkYDiskGetOutput2(b *testing.B) {
// 	// prepare for simulation
// 	err := exec.Command(SymExe, "setup").Run()
// 	if err != nil {
// 		b.Fatal("simulation prepare error")
// 	}
// 	out, err := exec.Command(SymExe, "start").Output()
// 	if err != nil {
// 		b.Fatal("simulation prepare error " + SymExe + err.Error() + string(out))
// 	}
// 	defer func() {
// 		err := exec.Command(SymExe, "stop").Run()
// 		if err != nil {
// 			b.Fatal("simulation stop error " + err.Error())
// 		}
// 	}()

// 	for range b.N {
// 		c := exec.Command(SymExe, "status")
// 		var stdout bytes.Buffer
// 		//stdout.Grow(256)
// 		c.Stdout = &stdout
// 		err := c.Run()
// 		st := stdout.Bytes()
// 		if err != nil || len(st) == 0 {
// 			b.Error(err)
// 		}
// 	}
// }

func BenchmarkEchoCmdOutput(b *testing.B) {
	for b.Loop() {
		st, err := exec.Command("echo", "test").Output()
		if err != nil || len(st) == 0 {
			b.Error(err)
		}
	}
}

func BenchmarkEchoCmdOutput2(b *testing.B) {
	for b.Loop() {
		c := exec.Command("echo", "test")
		var stdout bytes.Buffer
		c.Stdout = &stdout
		err := c.Run()
		st := stdout.Bytes()
		if err != nil || len(st) == 0 {
			b.Error(err)
		}
	}
}

func setChanged1(v *string, val string, c *bool) {
	*c = *c || *v != val
	*v = val
}

func TestSetChanged1(t *testing.T) {
	a := "none"
	c := false
	setChanged(&a, "idle", &c)
	require.True(t, c)
	require.Equal(t, "idle", a)
	b := "none"
	d := false
	setChanged1(&b, "idle", &d)
	require.Equal(t, a, b)
	require.Equal(t, c, d)
	c = false
	d = false
	setChanged(&a, "idle", &c)
	setChanged1(&b, "idle", &d)
	require.False(t, c)
	require.Equal(t, a, b)
	require.Equal(t, c, d)
}

func testChangedFunc(f func(v *string, val string, c *bool)) {
	a := "none"
	c := false
	f(&a, "idle", &c)
	f(&a, "idle", &c)
	f(&a, "none", &c)
	f(&a, "none", &c)
	f(&a, "idle", &c)
	f(&a, "idle", &c)
	f(&a, "none", &c)
	f(&a, "none", &c)
	f(&a, "idle", &c)
	f(&a, "idle", &c)
}

func BenchmarkSetChanged(b *testing.B) {
	for b.Loop() {
		testChangedFunc(setChanged)
	}
}

func BenchmarkSetChanged1(b *testing.B) {
	for b.Loop() {
		testChangedFunc(setChanged1)
	}
}

// update2 is original version with strings and not compiled regexp
func (val *YDvals) update2(out string) bool {
	val.Prev = val.Stat // store previous status but don't track changes of val.Prev
	changed := false    // track changes for values
	if out == "" {
		setChanged(&val.Stat, "none", &changed)
		if changed {
			val.Total, val.Used, val.Trash, val.Free = "", "", "", ""
			val.Prog, val.Err, val.ErrP, val.ChLast = "", "", "", true
			val.Last = []string{}
		}
		return changed
	}
	split := strings.Split(out, "Last synchronized items:")
	// Need to remove "Path to " as another "Path:" exists in case of access error
	split[0] = strings.Replace(split[0], "Path to ", "", 1)
	// Initialize map with keys that can be missed
	keys := map[string]string{"Sync": "", "Error": "", "Path": ""}
	// Take only first word in the phrase before ":"
	for _, s := range regexp.MustCompile(`\s*([^ ]+).*: (.*)`).FindAllStringSubmatch(split[0], -1) {
		if s[2][0] == byte('\'') {
			s[2] = s[2][1 : len(s[2])-1] // remove ' in the begging and at end
		}
		keys[s[1]] = s[2]
	}
	// map representation of switch_case clause
	for k, v := range map[string]*string{
		"Synchronization": &val.Stat,
		"Total":           &val.Total,
		"Used":            &val.Used,
		"Available":       &val.Free,
		"Trash":           &val.Trash,
		"Error":           &val.Err,
		"Path":            &val.ErrP,
		"Sync":            &val.Prog,
	} {
		setChanged(v, keys[k], &changed)
	}
	// Parse the "Last synchronized items" section (list of paths and files)
	val.ChLast = false // track last list changes separately
	if len(split) > 1 {
		f := regexp.MustCompile(`: '(.*)'\n`).FindAllStringSubmatch(split[1], -1)
		if len(f) != len(val.Last) {
			val.ChLast = true
			val.Last = []string{}
			for _, p := range f {
				val.Last = append(val.Last, p[1])
			}
		} else {
			for i, p := range f {
				setChanged(&val.Last[i], p[1], &val.ChLast)
			}
		}
	} else { // len(split) = 1 - there is no section with last sync. paths
		if len(val.Last) > 0 {
			val.Last = []string{}
			val.ChLast = true
		}
	}
	return changed || val.ChLast
}

// update1 used precompiled regexps
func (val *YDvals) update1(out string) bool {
	val.Prev = val.Stat // store previous status but don't track changes of val.Prev
	changed := false    // track changes for values
	if out == "" {
		setChanged(&val.Stat, "none", &changed)
		if changed {
			val.Total, val.Used, val.Trash, val.Free = "", "", "", ""
			val.Prog, val.Err, val.ErrP, val.ChLast = "", "", "", true
			val.Last = []string{}
		}
		return changed
	}
	split := strings.Split(out, "Last synchronized items:")
	// Initialize map with keys that can be missed
	keys := map[string]string{"Sync progress": "", "Error": "", "Path": ""}
	for _, s := range rPar.FindAllStringSubmatch(split[0], -1) {
		keys[s[1]] = s[2]
	}
	for k, v := range keys {
		switch k {
		case "Synchronization core status":
			setChanged(&val.Stat, v, &changed)
		case "Total":
			setChanged(&val.Total, v, &changed)
		case "Used":
			setChanged(&val.Used, v, &changed)
		case "Available":
			setChanged(&val.Free, v, &changed)
		case "Trash size":
			setChanged(&val.Trash, v, &changed)
		case "Error":
			setChanged(&val.Err, v, &changed)
		case "Path":
			if v != "" {
				setChanged(&val.ErrP, v[1:len(v)-1], &changed)
			} else {
				setChanged(&val.ErrP, "", &changed)
			}
		case "Sync progress":
			setChanged(&val.Prog, v, &changed)
		}
	}
	// Parse the "Last synchronized items" section (list of paths and files)
	val.ChLast = false // track last list changes separately
	if len(split) > 1 {
		f := rList.FindAllStringSubmatch(split[1], -1)
		if len(f) != len(val.Last) {
			val.ChLast = true
			val.Last = []string{}
			for _, p := range f {
				val.Last = append(val.Last, p[1])
			}
		} else {
			for i, p := range f {
				setChanged(&val.Last[i], p[1], &val.ChLast)
			}
		}
	} else { // len(split) = 1 - there is no section with last sync. paths
		if len(val.Last) > 0 {
			val.Last = []string{}
			val.ChLast = true
		}
	}
	return changed || val.ChLast
}
