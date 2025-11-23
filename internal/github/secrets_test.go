package github

import (
	"testing"
)

// TestDetectUnusedSecrets tests unused secret detection
func TestDetectUnusedSecrets(t *testing.T) {
	secrets := []Secret{
		{Name: "ACTIVE_SECRET", Scope: "org"},
		{Name: "UNUSED_SECRET", Scope: "org"},
		{Name: "REPO_SECRET", Scope: "repo", Repository: "owner/repo"},
	}

	workflowRefs := map[string][]string{
		"ACTIVE_SECRET": {".github/workflows/ci.yml"},
		"REPO_SECRET":   {".github/workflows/deploy.yml"},
		// UNUSED_SECRET is not referenced
	}

	usages := DetectUnusedSecrets(secrets, workflowRefs)

	if len(usages) != 3 {
		t.Errorf("Expected 3 usage entries, got %d", len(usages))
	}

	// Find UNUSED_SECRET
	var unusedUsage *SecretUsage
	for i := range usages {
		if usages[i].Name == "UNUSED_SECRET" {
			unusedUsage = &usages[i]
			break
		}
	}

	if unusedUsage == nil {
		t.Fatal("Expected to find UNUSED_SECRET usage")
	}

	if !unusedUsage.Unused {
		t.Error("Expected UNUSED_SECRET to be marked as unused")
	}

	if len(unusedUsage.ReferencedIn) != 0 {
		t.Errorf("Expected 0 references for UNUSED_SECRET, got %d",
			len(unusedUsage.ReferencedIn))
	}

	// Find ACTIVE_SECRET
	var activeUsage *SecretUsage
	for i := range usages {
		if usages[i].Name == "ACTIVE_SECRET" {
			activeUsage = &usages[i]
			break
		}
	}

	if activeUsage == nil {
		t.Fatal("Expected to find ACTIVE_SECRET usage")
	}

	if activeUsage.Unused {
		t.Error("Expected ACTIVE_SECRET to be marked as used")
	}

	if len(activeUsage.ReferencedIn) != 1 {
		t.Errorf("Expected 1 reference for ACTIVE_SECRET, got %d",
			len(activeUsage.ReferencedIn))
	}
}

// TestScanWorkflowForSecrets tests workflow scanning
func TestScanWorkflowForSecrets(t *testing.T) {
	workflowYAML := `
name: CI
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Run tests
        env:
          API_KEY: ${{ secrets.API_KEY }}
          DB_PASSWORD: ${{ secrets.DB_PASSWORD }}
          CUSTOM_VAR: ${{ vars.CUSTOM_VAR }}
        run: npm test
`

	refs := ScanWorkflowForSecrets(workflowYAML)

	expectedSecrets := map[string]bool{
		"API_KEY":     true,
		"DB_PASSWORD": true,
	}

	if len(refs) != len(expectedSecrets) {
		t.Errorf("Expected %d secret references, got %d: %v",
			len(expectedSecrets), len(refs), refs)
	}

	for _, ref := range refs {
		if !expectedSecrets[ref] {
			t.Errorf("Unexpected secret reference: %s", ref)
		}
	}

	// Should not include vars.CUSTOM_VAR (it's a variable, not a secret)
	for _, ref := range refs {
		if ref == "CUSTOM_VAR" {
			t.Error("Should not include variables (vars.*) in secret references")
		}
	}
}

// TestScanWorkflowMultipleFormats tests various secret reference formats
func TestScanWorkflowMultipleFormats(t *testing.T) {
	workflowYAML := `
name: Multi-format
jobs:
  deploy:
    steps:
      - run: echo "${{ secrets.TOKEN }}"
      - run: echo "${{secrets.ANOTHER_SECRET}}"
      - run: echo "${{ secrets.SPACED_SECRET }}"
      - run: echo "${{  secrets.EXTRA_SPACES  }}"
`

	refs := ScanWorkflowForSecrets(workflowYAML)

	expected := []string{"TOKEN", "ANOTHER_SECRET", "SPACED_SECRET", "EXTRA_SPACES"}

	if len(refs) != len(expected) {
		t.Errorf("Expected %d secrets, got %d: %v", len(expected), len(refs), refs)
	}

	// Convert to map for easier checking
	refMap := make(map[string]bool)
	for _, ref := range refs {
		refMap[ref] = true
	}

	for _, exp := range expected {
		if !refMap[exp] {
			t.Errorf("Expected to find secret '%s'", exp)
		}
	}
}

// TestGroupSecretsByScope tests grouping secrets
func TestGroupSecretsByScope(t *testing.T) {
	secrets := []Secret{
		{Name: "ORG_SECRET_1", Scope: "org"},
		{Name: "ORG_SECRET_2", Scope: "org"},
		{Name: "REPO_SECRET_1", Scope: "repo", Repository: "owner/repo1"},
		{Name: "REPO_SECRET_2", Scope: "repo", Repository: "owner/repo2"},
	}

	grouped := GroupSecretsByScope(secrets)

	if len(grouped["org"]) != 2 {
		t.Errorf("Expected 2 org secrets, got %d", len(grouped["org"]))
	}

	if len(grouped["repo"]) != 2 {
		t.Errorf("Expected 2 repo secrets, got %d", len(grouped["repo"]))
	}
}

// TestFindDuplicateSecrets tests duplicate detection across scopes
func TestFindDuplicateSecrets(t *testing.T) {
	secrets := []Secret{
		{Name: "API_KEY", Scope: "org"},
		{Name: "API_KEY", Scope: "repo", Repository: "owner/repo1"},
		{Name: "API_KEY", Scope: "repo", Repository: "owner/repo2"},
		{Name: "UNIQUE_SECRET", Scope: "org"},
	}

	duplicates := FindDuplicateSecrets(secrets)

	if len(duplicates) != 1 {
		t.Errorf("Expected 1 duplicate secret name, got %d", len(duplicates))
	}

	if duplicates[0].Name != "API_KEY" {
		t.Errorf("Expected API_KEY to be duplicate, got %s", duplicates[0].Name)
	}

	if duplicates[0].Count != 3 {
		t.Errorf("Expected API_KEY to appear 3 times, got %d", duplicates[0].Count)
	}
}
