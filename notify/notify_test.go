package notify

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDBusNotify(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
	// read icon
	p, err := os.Getwd()
	require.NoError(t, err)
	p, _ = path.Split(p)
	p += "/icons/img/logo.png"
	icon, err := os.ReadFile(p)
	require.NoError(t, err)

	n, err := New("appName", icon, true, -1)
	require.NoError(t, err)
	require.NotNil(t, n)
	defer n.Close()

	cap, err := n.Cap()
	require.NoError(t, err)
	require.NotEmpty(t, cap)

	n.Send("title", "message")
	time.Sleep(time.Second)
	n.Send("title1", "message1")
	time.Sleep(time.Second)
	n.Send("title2", "message2")
	time.Sleep(time.Second)
	n.replace = false
	n.Send("title3", "message3")
	time.Sleep(time.Second)
	n.Send("title4", "message4")
	time.Sleep(time.Second)
}
