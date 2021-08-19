package icons

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Data(t *testing.T) {
	assert.Equal(t, 719, len(lightBusy1))
	assert.Equal(t, 709, len(lightBusy2))
	assert.Equal(t, 722, len(lightBusy3))
	assert.Equal(t, 724, len(lightBusy4))
	assert.Equal(t, 711, len(lightBusy5))
	assert.Equal(t, 711, len(lightError))
	assert.Equal(t, 701, len(lightIdle))
	assert.Equal(t, 720, len(lightPause))
	assert.Equal(t, 1063, len(darkBusy1))
	assert.Equal(t, 1083, len(darkBusy2))
	assert.Equal(t, 1080, len(darkBusy3))
	assert.Equal(t, 1076, len(darkBusy4))
	assert.Equal(t, 1056, len(darkBusy5))
	assert.Equal(t, 956, len(darkError))
	assert.Equal(t, 1072, len(darkIdle))
	assert.Equal(t, 1069, len(darkPause))
	assert.Equal(t, 14251, len(yd128))
}
