package cmd

import (
	"testing"
)

func TestRootCmd(t *testing.T) {
	// Test that root command can be created
	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}

	if rootCmd.Use != "gh-sweep" {
		t.Errorf("Expected Use to be 'gh-sweep', got '%s'", rootCmd.Use)
	}
}

func TestVersionInfo(t *testing.T) {
	// Test that version variables exist
	if version == "" {
		t.Error("version is empty")
	}
}
