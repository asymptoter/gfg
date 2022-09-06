package c

import "testing"

func TestC(t *testing.T) {
	if C != "c" {
		t.Fatal("not c")
	}
}
