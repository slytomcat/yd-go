package icons

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Data(t *testing.T) {
	assert.NotEmpty(t, lightBusy1)
	assert.NotEmpty(t, lightBusy2)
	assert.NotEmpty(t, lightBusy3)
	assert.NotEmpty(t, lightBusy4)
	assert.NotEmpty(t, lightBusy5)
	assert.NotEmpty(t, lightError)
	assert.NotEmpty(t, lightIdle)
	assert.NotEmpty(t, lightPause)
	assert.NotEmpty(t, darkBusy1)
	assert.NotEmpty(t, darkBusy2)
	assert.NotEmpty(t, darkBusy3)
	assert.NotEmpty(t, darkBusy4)
	assert.NotEmpty(t, darkBusy5)
	assert.NotEmpty(t, darkError)
	assert.NotEmpty(t, darkIdle)
	assert.NotEmpty(t, darkPause)
	assert.NotEmpty(t, logo)
}
