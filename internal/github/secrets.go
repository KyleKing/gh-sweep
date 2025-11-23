package github

import (
	"fmt"
	"regexp"
)

// Secret represents a GitHub Actions secret
type Secret struct {
	Name       string
	Scope      string // "org" or "repo"
	Repository string // Empty for org secrets
	CreatedAt  string
	UpdatedAt  string
}

type secretsResponse struct {
	Secrets []struct {
		Name      string `json:"name"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	} `json:"secrets"`
}

// ListOrgSecrets lists organization-level secrets
func (c *Client) ListOrgSecrets(org string) ([]Secret, error) {
	var response secretsResponse
	path := fmt.Sprintf("orgs/%s/actions/secrets", org)

	if err := c.Get(path, &response); err != nil {
		return nil, fmt.Errorf("failed to list org secrets: %w", err)
	}

	secrets := make([]Secret, len(response.Secrets))
	for i, s := range response.Secrets {
		secrets[i] = Secret{
			Name:      s.Name,
			Scope:     "org",
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
		}
	}

	return secrets, nil
}

// ListRepoSecrets lists repository-level secrets
func (c *Client) ListRepoSecrets(owner, repo string) ([]Secret, error) {
	var response secretsResponse
	path := fmt.Sprintf("repos/%s/%s/actions/secrets", owner, repo)

	if err := c.Get(path, &response); err != nil {
		return nil, fmt.Errorf("failed to list repo secrets: %w", err)
	}

	secrets := make([]Secret, len(response.Secrets))
	for i, s := range response.Secrets {
		secrets[i] = Secret{
			Name:       s.Name,
			Scope:      "repo",
			Repository: fmt.Sprintf("%s/%s", owner, repo),
			CreatedAt:  s.CreatedAt,
			UpdatedAt:  s.UpdatedAt,
		}
	}

	return secrets, nil
}

// SecretUsage tracks secret usage in workflows
type SecretUsage struct {
	Name         string
	Scope        string
	Repository   string
	ReferencedIn []string // Workflow files that reference this secret
	Unused       bool
}

// DetectUnusedSecrets compares secrets against workflow references
func DetectUnusedSecrets(secrets []Secret, workflowRefs map[string][]string) []SecretUsage {
	usages := []SecretUsage{}

	for _, secret := range secrets {
		usage := SecretUsage{
			Name:       secret.Name,
			Scope:      secret.Scope,
			Repository: secret.Repository,
		}

		// Check if secret is referenced
		if refs, ok := workflowRefs[secret.Name]; ok {
			usage.ReferencedIn = refs
			usage.Unused = false
		} else {
			usage.Unused = true
		}

		usages = append(usages, usage)
	}

	return usages
}

// ScanWorkflowForSecrets extracts secret references from workflow YAML
// Pure function: parses YAML content for secrets.* references
func ScanWorkflowForSecrets(workflowContent string) []string {
	// Match ${{ secrets.SECRET_NAME }} pattern (with optional spaces)
	pattern := regexp.MustCompile(`\${{\s*secrets\.([A-Z0-9_]+)\s*}}`)
	matches := pattern.FindAllStringSubmatch(workflowContent, -1)

	// Deduplicate secret names
	secretSet := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			secretSet[match[1]] = true
		}
	}

	// Convert to slice
	secrets := make([]string, 0, len(secretSet))
	for secret := range secretSet {
		secrets = append(secrets, secret)
	}

	return secrets
}

// GroupSecretsByScope groups secrets by their scope (org/repo)
// Pure function: creates grouped map
func GroupSecretsByScope(secrets []Secret) map[string][]Secret {
	grouped := make(map[string][]Secret)

	for _, secret := range secrets {
		grouped[secret.Scope] = append(grouped[secret.Scope], secret)
	}

	return grouped
}

// DuplicateSecret represents a secret name that appears multiple times
type DuplicateSecret struct {
	Name   string
	Count  int
	Scopes []string // List of scopes where it appears
	Repos  []string // List of repositories (for repo-scoped secrets)
}

// FindDuplicateSecrets identifies secret names that appear in multiple scopes/repos
// Pure function: analyzes secret list for duplicates
func FindDuplicateSecrets(secrets []Secret) []DuplicateSecret {
	// Track occurrences
	occurrences := make(map[string]*DuplicateSecret)

	for _, secret := range secrets {
		if dup, exists := occurrences[secret.Name]; exists {
			dup.Count++
			// Add scope if not already present
			if !contains(dup.Scopes, secret.Scope) {
				dup.Scopes = append(dup.Scopes, secret.Scope)
			}
			// Add repo if repo-scoped
			if secret.Scope == "repo" && !contains(dup.Repos, secret.Repository) {
				dup.Repos = append(dup.Repos, secret.Repository)
			}
		} else {
			occurrences[secret.Name] = &DuplicateSecret{
				Name:   secret.Name,
				Count:  1,
				Scopes: []string{secret.Scope},
				Repos:  []string{},
			}
			if secret.Scope == "repo" {
				occurrences[secret.Name].Repos = []string{secret.Repository}
			}
		}
	}

	// Filter to only duplicates (count > 1)
	duplicates := make([]DuplicateSecret, 0)
	for _, dup := range occurrences {
		if dup.Count > 1 {
			duplicates = append(duplicates, *dup)
		}
	}

	return duplicates
}
