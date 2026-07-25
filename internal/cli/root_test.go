package cli

import (
	"testing"
)

func TestRootCmd(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}

	if rootCmd.Use != "gh-sweep" {
		t.Errorf("Expected Use to be 'gh-sweep', got '%s'", rootCmd.Use)
	}
}
