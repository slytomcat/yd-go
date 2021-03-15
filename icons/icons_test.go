package icons

import "testing"

func Test_Data01(t *testing.T) {
	if len(darkBusy1) == 0 || len(darkBusy2) == 0 || len(darkBusy3) == 0 || len(darkBusy4) == 0 || len(darkBusy5) == 0 {
		t.Error("len(darkBusy) = 0")
	}
}
