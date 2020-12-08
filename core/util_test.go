package core

import (
	"testing"
)

func Test_RandomInt(t *testing.T) {
	c := 1
	for c <= 1000 {
		w := RandomInt(100, 1000)
		if w > 1000 || w < 100 {
			t.Error("wait time is out of range (100 - 1000)")
		}
		c++
	}
}
