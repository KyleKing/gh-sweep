package github

import "fmt"

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
