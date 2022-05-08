package notify

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDBusNotify(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
	icon := "dialog-information"
	n, err := New("appname", "", true, -1)
	require.NoError(t, err)
	require.NotNil(t, n)

	cap := n.Cap()
	require.NotEmpty(t, cap)

	n.Send(icon, "title", "message")
	time.Sleep(time.Second)
	n.Send("dialog-error", "title1", "message1")
	time.Sleep(time.Second)
	n.Send("dialog-warning", "title2", "message2")
	time.Sleep(time.Second)
	n.Send(icon, "title3", "message3")
	time.Sleep(time.Second)
	n.Send("", "title4", "message4")

}
