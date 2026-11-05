package github

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const generationDir = "generations"

// branchGeneration returns the repo's branch-list cache generation, or "" when
// the client has no cache or the branch list has never been invalidated.
func (c *Client) branchGeneration(owner, repo string) string {
	if c.cacheDir == "" {
		return ""
	}

	data, err := os.ReadFile(c.generationPath(owner, repo))
	if err != nil {
		return ""
	}

	return string(data)
}

// pruneBranchList makes the next branch listing for the repo miss the HTTP
// cache. Entries are keyed by URL, so a new generation in the listing URL
// strands the old entries until their TTL expires, and nothing else the cache
// holds is touched.
func (c *Client) pruneBranchList(owner, repo string) error {
	if c.cacheDir == "" {
		return nil
	}

	path := c.generationPath(owner, repo)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating cache generation dir: %w", err)
	}

	generation := strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := os.WriteFile(path, []byte(generation), 0o600); err != nil {
		return fmt.Errorf("writing cache generation: %w", err)
	}

	return nil
}

func (c *Client) generationPath(owner, repo string) string {
	return filepath.Join(c.cacheDir, generationDir, owner+"__"+repo+".branches")
}
