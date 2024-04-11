package c

import "testing"

func TestC(t *testing.T) {
	if C != "cd" {
		t.Error("C should be `cd`")
	}
}
