package scanner

import (
	"testing"
)

func TestScan(t *testing.T) {
	_, err := NewScanner(nil)
	if err != nil {
		t.Error(err)
		return
	}
}
