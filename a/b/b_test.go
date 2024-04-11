package b

import "testing"

func TestB(t *testing.T) {
	if B != "bcd" {
		t.Error("B should be equal to `bcd`")
	}
}
