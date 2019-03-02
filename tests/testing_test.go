package testtest

import (
	"testing"
)

func TestTesting(t *testing.T) {
	if true == false {
		t.Fatalf("Wtf")
	}
}
