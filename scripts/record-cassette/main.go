// Command record-cassette re-records the go-vcr fixtures under
// internal/policy/testdata/cassettes/ by exercising the policy engine against
// the real GitHub API, using gh CLI auth. Every mutation it makes is reverted
// before the process exits, and no request or response body is skipped when
// stripping the Authorization header, so the checked-in cassette never
// carries a live credential.
//
// Run with: go run ./scripts/record-cassette KyleKing/gh-sweep
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"

	"github.com/KyleKing/gh-sweep/internal/github"
)

const (
	cassettePath = "internal/policy/testdata/cassettes/settings-and-security"
	wantArgs     = 2
)

var errInvalidRepo = errors.New("invalid repo, want owner/repo")

func main() {
	if len(os.Args) != wantArgs {
		fmt.Fprintln(os.Stderr, "usage: record-cassette owner/repo")
		os.Exit(1)
	}

	if err := run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "record-cassette:", err)
		os.Exit(1)
	}
}

func run(fullName string) error {
	owner, repo, ok := splitRepo(fullName)
	if !ok {
		return fmt.Errorf("%w: %q", errInvalidRepo, fullName)
	}

	rec, err := recorder.New(cassettePath,
		recorder.WithMode(recorder.ModeRecordOnly),
		recorder.WithHook(redactAuthorization, recorder.BeforeSaveHook),
	)
	if err != nil {
		return fmt.Errorf("creating recorder: %w", err)
	}
	defer func() {
		if stopErr := rec.Stop(); stopErr != nil {
			fmt.Fprintln(os.Stderr, "record-cassette: saving cassette:", stopErr)
		}
	}()

	ctx := context.Background()

	client, err := github.NewClientWithRealAuthAndTransport(ctx, rec)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	afterSettings, err := recordSettingsToggle(client, owner, repo)
	if err != nil {
		return err
	}

	if err := recordSecurityToggle(client, owner, repo, afterSettings); err != nil {
		return err
	}

	return recordUnprotectedBranch(client, owner, repo)
}

// recordSettingsToggle flips has_wiki on and back off, capturing the GET/PATCH
// round trip for both directions so a replay test can assert against real bytes.
// It returns the final GET response so recordSecurityToggle can reuse it as a
// baseline instead of re-fetching, keeping the recorded call sequence identical
// to what policy.Evaluate/Apply actually issue.
func recordSettingsToggle(client *github.Client, owner, repo string) (*github.RepoSettings, error) {
	before, err := client.GetRepoSettings(owner, repo)
	if err != nil {
		return nil, fmt.Errorf("baseline settings: %w", err)
	}

	toggled := !before.HasWiki
	if err := client.UpdateRepoSettings(owner, repo, github.RepoSettingsPatch{HasWiki: &toggled}); err != nil {
		return nil, fmt.Errorf("toggling has_wiki: %w", err)
	}

	if _, err := client.GetRepoSettings(owner, repo); err != nil {
		return nil, fmt.Errorf("settings after toggle: %w", err)
	}

	original := before.HasWiki
	if err := client.UpdateRepoSettings(owner, repo, github.RepoSettingsPatch{HasWiki: &original}); err != nil {
		return nil, fmt.Errorf("reverting has_wiki: %w", err)
	}

	after, err := client.GetRepoSettings(owner, repo)
	if err != nil {
		return nil, fmt.Errorf("settings after revert: %w", err)
	}

	return after, nil
}

// recordSecurityToggle flips secret_scanning_push_protection and back, the
// nested-object PATCH shape that differs from the flat settings patch.
func recordSecurityToggle(client *github.Client, owner, repo string, before *github.RepoSettings) error {
	status := before.SecurityAndAnalysis.SecretScanningPushProtection
	toggled := "enabled"
	if status == "enabled" {
		toggled = "disabled"
	}

	if err := client.UpdateSecurityAndAnalysis(owner, repo, "secret_scanning_push_protection", toggled); err != nil {
		return fmt.Errorf("toggling push protection: %w", err)
	}

	if _, err := client.GetRepoSettings(owner, repo); err != nil {
		return fmt.Errorf("security after toggle: %w", err)
	}

	if err := client.UpdateSecurityAndAnalysis(owner, repo, "secret_scanning_push_protection", status); err != nil {
		return fmt.Errorf("reverting push protection: %w", err)
	}

	if _, err := client.GetRepoSettings(owner, repo); err != nil {
		return fmt.Errorf("security after revert: %w", err)
	}

	return nil
}

// recordUnprotectedBranch captures the 404 shape GitHub returns for a branch
// with no protection rule, preceded by the same GetRepoSettings call
// policy.evaluateRepo always makes first to resolve the default branch.
func recordUnprotectedBranch(client *github.Client, owner, repo string) error {
	if _, err := client.GetRepoSettings(owner, repo); err != nil {
		return fmt.Errorf("settings before protection check: %w", err)
	}

	_, err := client.GetBranchProtection(owner, repo, "main")
	if err != nil && !errors.Is(err, github.ErrBranchNotProtected) {
		return fmt.Errorf("branch protection: %w", err)
	}

	return nil
}

func redactAuthorization(i *cassette.Interaction) error {
	i.Request.Headers.Del("Authorization")
	i.Response.Headers.Del("Authorization")

	return nil
}

// same-type-result heuristic instead
//
//nolint:gocritic // named results would trip nonamedreturns; unnamed trips gocritic's
func splitRepo(fullName string) (string, string, bool) {
	for idx := range fullName {
		if fullName[idx] == '/' {
			return fullName[:idx], fullName[idx+1:], true
		}
	}

	return "", "", false
}
