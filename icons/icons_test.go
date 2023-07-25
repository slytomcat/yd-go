package icons

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockIcon struct {
	icon []byte
	mu   sync.Mutex
}

func (m *mockIcon) set(icon []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.icon = icon
}

func (m *mockIcon) get() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.icon
}

var mi mockIcon

func TestNewIcon(t *testing.T) {
	i := NewIcon("dark", mi.set)
	require.NotNil(t, i)
	defer i.CleanUp()
	assert.Equal(t, darkPause, mi.get())
	assert.Equal(t, darkError, i.errorIcon)
	assert.Equal(t, darkIdle, i.idleIcon)
	assert.Equal(t, darkPause, i.pauseIcon)
	assert.Equal(t, [5][]byte{darkBusy1, darkBusy2, darkBusy3, darkBusy4, darkBusy5}, i.busyIcons)
}

func TestSetTheme(t *testing.T) {
	i := NewIcon("dark", mi.set)
	require.NotNil(t, i)
	defer i.CleanUp()
	assert.Equal(t, darkPause, mi.get())
	i.SetTheme("light")
	assert.Equal(t, darkPause, mi.get()) // current icon should not be changed if Set() was not called after NewIcon() as status is still unknown
	assert.Equal(t, lightError, i.errorIcon)
	assert.Equal(t, lightIdle, i.idleIcon)
	assert.Equal(t, lightPause, i.pauseIcon)
	assert.Equal(t, [5][]byte{lightBusy1, lightBusy2, lightBusy3, lightBusy4, lightBusy5}, i.busyIcons)
	i.Set("idle")
	assert.Equal(t, lightIdle, mi.get())
	i.SetTheme("dark")
	assert.Equal(t, darkIdle, mi.get()) // after a call of Set(), the SetTheme() should change current icon
}

func TestSet(t *testing.T) {
	i := NewIcon("dark", mi.set)
	require.NotNil(t, i)
	defer i.CleanUp()
	i.Set("error")
	assert.Equal(t, darkError, mi.get())
	i.Set("idle")
	assert.Equal(t, darkIdle, mi.get())
	i.Set("none")
	assert.Equal(t, darkPause, mi.get())
	i.Set("busy")
	assert.Equal(t, darkBusy1, mi.get())
}

func TestAnimation(t *testing.T) {
	interval = 10 * time.Millisecond
	tick := time.Millisecond
	waitFor := interval + 5*tick
	event := func(i []byte) func() bool {
		return func() bool { return bytes.Equal(mi.get(), i) }
	}

	i := NewIcon("dark", mi.set)
	require.NotNil(t, i)
	defer i.CleanUp()
	i.Set("index")
	assert.Equal(t, darkBusy1, mi.get())
	assert.Eventually(t, event(darkBusy2), waitFor, tick)
	assert.Eventually(t, event(darkBusy3), waitFor, tick)
	assert.Eventually(t, event(darkBusy4), waitFor, tick)
	assert.Eventually(t, event(darkBusy5), waitFor, tick)
	assert.Eventually(t, event(darkBusy1), waitFor, tick)
}

func TestWrongTheme(t *testing.T) {
	require.Panics(t, func() {
		_ = NewIcon("wrong", mi.set)
	})
}

func TestDoubleCleanUp(t *testing.T) {
	i := NewIcon("dark", mi.set)
	require.NotNil(t, i)
	i.CleanUp()
	require.NotPanics(t, i.CleanUp)
}
