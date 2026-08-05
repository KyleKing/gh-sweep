package github

import "fmt"

// ImmutableReleases represents a repository's release-immutability status.
type ImmutableReleases struct {
	Repository      string
	Enabled         bool
	EnforcedByOwner bool
}

type immutableReleasesResponse struct {
	Enabled         bool `json:"enabled"`
	EnforcedByOwner bool `json:"enforced_by_owner"`
}

// GetImmutableReleases retrieves whether release immutability is enabled for a repo.
func (c *Client) GetImmutableReleases(owner, repo string) (*ImmutableReleases, error) {
	var response immutableReleasesResponse
	path := fmt.Sprintf("repos/%s/%s/immutable-releases", owner, repo)

	if err := c.Get(path, &response); err != nil {
		return nil, fmt.Errorf("failed to get immutable releases setting: %w", err)
	}

	return &ImmutableReleases{
		Repository:      fmt.Sprintf("%s/%s", owner, repo),
		Enabled:         response.Enabled,
		EnforcedByOwner: response.EnforcedByOwner,
	}, nil
}

// SetImmutableReleases enables or disables release immutability for a repo.
func (c *Client) SetImmutableReleases(owner, repo string, enabled bool) error {
	path := fmt.Sprintf("repos/%s/%s/immutable-releases", owner, repo)

	if enabled {
		if err := c.Put(path, map[string]any{}, nil); err != nil {
			return fmt.Errorf("failed to enable immutable releases: %w", err)
		}

		return nil
	}

	if err := c.Delete(path, nil); err != nil {
		return fmt.Errorf("failed to disable immutable releases: %w", err)
	}

	return nil
}
