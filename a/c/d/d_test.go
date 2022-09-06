package d

import "testing"

func TestD(t *testing.T) {
	if D != "d" {
		t.Fatal("not d")
	}
}
