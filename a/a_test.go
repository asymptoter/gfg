package a

import "testing"

func TestB(t *testing.T) {
	if B() != "bcd" {
		t.Error("B() should be equal to `bcd`")
	}
}

func TestC(t *testing.T) {
	if C() != "cd" {
		t.Error("C() should be equal to `cd`")
	}
}
