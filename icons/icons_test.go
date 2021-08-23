package icons

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type fakeIcon struct {
	icon []byte
	mu   sync.Mutex
}

func (f *fakeIcon) set(icon []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.icon = icon
}

func (f *fakeIcon) get() []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.icon
}

var fi fakeIcon

func TestNewIcon(t *testing.T) {
	i := NewIcon("dark", fi.set)
	assert.NotNil(t, i)
	defer i.CleanUp()
	assert.NotNil(t, i)
	assert.Equal(t, darkPause, fi.get())
	assert.Equal(t, darkError, i.errorIcon)
	assert.Equal(t, darkIdle, i.idleIcon)
	assert.Equal(t, darkPause, i.pauseIcon)
	assert.Equal(t, [5][]byte{darkBusy1, darkBusy2, darkBusy3, darkBusy4, darkBusy5}, i.busyIcons)
}

func TestSetTheme(t *testing.T) {
	i := NewIcon("dark", fi.set)
	assert.NotNil(t, i)
	defer i.CleanUp()
	assert.Equal(t, darkPause, fi.get())
	i.SetTheme("light")
	assert.Equal(t, darkPause, fi.get()) // current icon should not be changed if Set() was not called after NewIcon()
	assert.Equal(t, lightError, i.errorIcon)
	assert.Equal(t, lightIdle, i.idleIcon)
	assert.Equal(t, lightPause, i.pauseIcon)
	assert.Equal(t, [5][]byte{lightBusy1, lightBusy2, lightBusy3, lightBusy4, lightBusy5}, i.busyIcons)
	i.Set("idle")
	assert.Equal(t, lightIdle, fi.get())
	i.SetTheme("dark")
	assert.Equal(t, darkIdle, fi.get()) // after a call of Set(), the SetTheme() should change current icon
}

func TestSet(t *testing.T) {
	i := NewIcon("dark", fi.set)
	assert.NotNil(t, i)
	defer i.CleanUp()
	i.Set("error")
	assert.Equal(t, darkError, fi.get())
	i.Set("idle")
	assert.Equal(t, darkIdle, fi.get())
	i.Set("none")
	assert.Equal(t, darkPause, fi.get())
	i.Set("busy")
	assert.Equal(t, darkBusy1, fi.get())
}

func TestAnimation(t *testing.T) {
	interval = 10 * time.Millisecond
	tick := time.Millisecond
	waitFor := interval + 5*tick
	event := func(i []byte) func() bool {
		return func() bool { return bytes.Equal(fi.get(), i) }
	}

	i := NewIcon("dark", fi.set)
	assert.NotNil(t, i)
	defer i.CleanUp()
	i.Set("index")
	assert.Equal(t, darkBusy1, fi.get())
	assert.Eventually(t, event(darkBusy2), waitFor, tick)
	assert.Eventually(t, event(darkBusy3), waitFor, tick)
	assert.Eventually(t, event(darkBusy4), waitFor, tick)
	assert.Eventually(t, event(darkBusy5), waitFor, tick)
	assert.Eventually(t, event(darkBusy1), waitFor, tick)
}

func TestWrongTheme(t *testing.T) {
	defer func() {
		assert.NotNil(t, recover())
	}()

	_ = NewIcon("wrong", fi.set)

	assert.FailNow(t, "we should not get here")
}

func TestDubleCleanUp(t *testing.T) {
	i := NewIcon("dark", fi.set)
	assert.NotNil(t, i)
	i.CleanUp()
	defer func() {
		assert.NotNil(t, recover())
	}()

	i.CleanUp()

	assert.FailNow(t, "we should not get here")
}
