//go:build wasm

package tests

import (
	"github.com/veltylabs/work_schedule/seed"
	"testing"
)

func TestDemo_Valid(t *testing.T) {
	d, err := seed.Demo()
	if err != nil {
		t.Fatalf("seed.Demo() returned err: %v", err)
	}
	if err := d.Validate(); err != nil {
		t.Fatalf("d.Validate() failed: %v", err)
	}
}
