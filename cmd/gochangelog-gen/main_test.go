package main

import (
	"testing"
)

func TestVersionVariablesExist(t *testing.T) {
	// Ensures that the linker variables are declared and have sensible defaults.
	if version == "" {
		t.Error("version variable should not be empty")
	}
	if buildTime == "" {
		t.Error("buildTime variable should not be empty")
	}
}
