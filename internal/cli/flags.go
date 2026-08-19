package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/KyleKing/gh-sweep/internal/config"
	"github.com/KyleKing/gh-sweep/internal/github"
)

const (
	repoPartsCount = 2
	confirmYes     = "yes"
	formatJSON     = "json"
	formatMarkdown = "markdown"
)

func boolFlag(cmd *cobra.Command, name string) bool {
	value, err := cmd.Flags().GetBool(name)
	if err != nil {
		return false
	}

	return value
}

func intFlag(cmd *cobra.Command, name string) int {
	value, err := cmd.Flags().GetInt(name)
	if err != nil {
		return 0
	}

	return value
}

func stringFlag(cmd *cobra.Command, name string) string {
	value, err := cmd.Flags().GetString(name)
	if err != nil {
		return ""
	}

	return value
}

func stringSliceFlag(cmd *cobra.Command, name string) []string {
	value, err := cmd.Flags().GetStringSlice(name)
	if err != nil {
		return nil
	}

	return value
}

func splitRepo(repo string) (string, string, bool) {
	parts := strings.SplitN(repo, "/", repoPartsCount)
	if len(parts) != repoPartsCount || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}

	return parts[0], parts[1], true
}

func resolveNamespace(cmd *cobra.Command, client *github.Client, cfg *config.Config) (string, error) {
	namespace := stringFlag(cmd, "namespace")
	if namespace == "" {
		namespace = stringFlag(cmd, "org")
	}
	if namespace == "" {
		namespace = cfg.DefaultOrg
	}
	if namespace != "" {
		return namespace, nil
	}

	username, err := client.GetAuthenticatedUser()
	if err != nil {
		return "", fmt.Errorf("failed to get authenticated user: %w", err)
	}

	return username, nil
}

func confirmTypedYes(prompt string) bool {
	fmt.Print(prompt)

	reader := bufio.NewReader(os.Stdin)

	line, err := reader.ReadString('\n')
	if err != nil || strings.TrimSpace(line) != confirmYes {
		return false
	}

	return true
}

func writeOutput(
	outputPath, format string,
	toJSON func() ([]byte, error),
	toMarkdown func() string,
	toTable func(*strings.Builder),
) {
	var output string

	switch format {
	case formatJSON:
		data, err := toJSON()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to marshal JSON: %v\n", err)
			os.Exit(1)
		}
		output = string(data)

	case formatMarkdown:
		output = toMarkdown()

	default:
		var b strings.Builder
		toTable(&b)
		output = b.String()
	}

	if outputPath == "" {
		fmt.Print(output)
		return
	}

	if err := os.WriteFile(outputPath, []byte(output), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to write output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Output written to: %s\n", outputPath)
}
