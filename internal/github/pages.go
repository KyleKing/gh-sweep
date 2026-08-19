package github

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// PagesInfo is a repository's GitHub Pages configuration.
type PagesInfo struct {
	Repository     string
	CNAME          string
	HTMLURL        string
	HTTPSEnforced  bool
	Status         string
	DomainVerified bool
}

type pagesResponse struct {
	CNAME                string `json:"cname"`
	HTMLURL              string `json:"html_url"`
	HTTPSEnforced        bool   `json:"https_enforced"`
	Status               string `json:"status"`
	ProtectedDomainState string `json:"protected_domain_state"`
}

// GetPagesInfo fetches a repo's GitHub Pages configuration. It returns
// nil, nil when Pages isn't enabled for the repo (a 404 from the API).
func (c *Client) GetPagesInfo(owner, repo string) (*PagesInfo, error) {
	path := fmt.Sprintf("repos/%s/%s/pages", owner, repo)

	var response pagesResponse
	if err := c.Get(path, &response); err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get pages info: %w", err)
	}

	return &PagesInfo{
		Repository:     fmt.Sprintf("%s/%s", owner, repo),
		CNAME:          response.CNAME,
		HTMLURL:        response.HTMLURL,
		HTTPSEnforced:  response.HTTPSEnforced,
		Status:         response.Status,
		DomainVerified: response.ProtectedDomainState == "" || response.ProtectedDomainState == "verified",
	}, nil
}

type contentsResponse struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

// GetCNAMEFile reads a repo's root CNAME file, returning "" when the repo
// has none. A CNAME file can outlive Pages being disabled, which is the
// subdomain-takeover signal: DNS still points at GitHub while nothing serves
// the domain from this repo anymore.
func (c *Client) GetCNAMEFile(owner, repo string) (string, error) {
	path := fmt.Sprintf("repos/%s/%s/contents/CNAME", owner, repo)

	var response contentsResponse
	if err := c.Get(path, &response); err != nil {
		if strings.Contains(err.Error(), "404") {
			return "", nil
		}

		return "", fmt.Errorf("failed to read CNAME file: %w", err)
	}

	if response.Encoding != "base64" {
		return "", fmt.Errorf("unexpected CNAME file encoding: %s", response.Encoding)
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(response.Content, "\n", ""))
	if err != nil {
		return "", fmt.Errorf("failed to decode CNAME file: %w", err)
	}

	return strings.TrimSpace(string(decoded)), nil
}
