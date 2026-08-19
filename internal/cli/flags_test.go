package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func testFlagCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("verbose", false, "")
	cmd.Flags().Int("limit", 5, "")
	cmd.Flags().String("name", "default", "")
	cmd.Flags().StringSlice("tags", nil, "")

	return cmd
}

func TestFlagHelpers(t *testing.T) {
	t.Parallel()

	cmd := testFlagCommand()

	if got := boolFlag(cmd, "verbose"); got {
		t.Errorf("boolFlag() = %v, want false", got)
	}
	if got := boolFlag(cmd, "missing"); got {
		t.Errorf("boolFlag(missing) = %v, want false", got)
	}

	if got := intFlag(cmd, "limit"); got != 5 {
		t.Errorf("intFlag() = %d, want 5", got)
	}
	if got := intFlag(cmd, "missing"); got != 0 {
		t.Errorf("intFlag(missing) = %d, want 0", got)
	}

	if got := stringFlag(cmd, "name"); got != "default" {
		t.Errorf("stringFlag() = %q, want default", got)
	}
	if got := stringFlag(cmd, "missing"); got != "" {
		t.Errorf("stringFlag(missing) = %q, want empty", got)
	}

	if err := cmd.Flags().Set("tags", "a,b"); err != nil {
		t.Fatalf("failed to set tags flag: %v", err)
	}
	if got := stringSliceFlag(cmd, "tags"); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("stringSliceFlag() = %v, want [a b]", got)
	}
	if got := stringSliceFlag(cmd, "missing"); got != nil {
		t.Errorf("stringSliceFlag(missing) = %v, want nil", got)
	}
}
