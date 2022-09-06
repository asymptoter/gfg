package b

import "testing"

func TestB(t *testing.T) {
	if B != "bc" {
		t.Fatal("not bc")
	}
}
