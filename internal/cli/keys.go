package cli

import (
	"errors"
	"fmt"
	"time"

	"github.com/kyleking/aragonite/keyusage"
	"github.com/spf13/cobra"

	"github.com/KyleKing/gh-sweep/internal/tui"
)

const usageApp = "gh-sweep"

var errNoUsageLog = errors.New("no usage log")

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Report which TUI keys you press, never press, or press where nothing is bound",
	Long: `Read the key usage log back against the TUI's key bindings.

Keys are logged only from the home menu and the views opened from it, into
$GH_SWEEP_USAGE_DIR, else ~/Sync/.gh-sweep/usage when ~/Sync exists. Text typed
into a search or edit box is never logged. With neither location, nothing is
recorded and this command has nothing to report.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		dir := keyusage.Dir(usageApp)
		if dir == "" {
			return fmt.Errorf("%w: set %s or create ~/Sync", errNoUsageLog, keyusage.EnvVar(usageApp))
		}

		km := tui.Keymap()

		u, err := keyusage.Load(dir, km)
		if err != nil {
			return fmt.Errorf("reading the usage log: %w", err)
		}

		if err := keyusage.Write(cmd.OutOrStdout(), u, km, time.Now().Add(-tui.StaleAfter)); err != nil {
			return fmt.Errorf("writing the key report: %w", err)
		}

		return nil
	},
}

func init() { rootCmd.AddCommand(keysCmd) }
