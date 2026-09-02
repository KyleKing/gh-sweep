package github

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ErrRulesetNotFound means the repo has no ruleset with the requested name.
var ErrRulesetNotFound = errors.New("ruleset not found")

// Ruleset is a repository ruleset flattened from GitHub's {type, parameters}
// rule array into the subset gh-sweep manages. Rules the policy does not model
// survive a round trip through Unmanaged, so updating a ruleset never silently
// drops a rule gh-sweep cannot express.
type Ruleset struct {
	ID                   int
	Name                 string
	Target               string
	Enforcement          string
	IncludeRefs          []string
	ExcludeRefs          []string
	BlockDeletion        bool
	BlockForcePush       bool
	RequireLinearHistory bool
	RequiredStatusChecks []string
	PullRequest          *PullRequestRule
	BypassActors         []json.RawMessage
	Unmanaged            []json.RawMessage
}

// PullRequestRule is a ruleset's pull_request rule. Its presence is what
// requires a PR at all; RequiredApprovals of 0 requires the PR without
// requiring an approval, which classic branch protection cannot express.
type PullRequestRule struct {
	RequiredApprovals              int
	RequireCodeOwnerReview         bool
	RequireLastPushApproval        bool
	DismissStaleReviewsOnPush      bool
	RequiredReviewThreadResolution bool
	AllowedMergeMethods            []string
}

const (
	ruleDeletion         = "deletion"
	ruleNonFastForward   = "non_fast_forward"
	ruleLinearHistory    = "required_linear_history"
	rulePullRequest      = "pull_request"
	ruleStatusChecks     = "required_status_checks"
	rulesetTargetBranch  = "branch"
	modeledRuleTypes     = 5
	rulesetDefaultBranch = "~DEFAULT_BRANCH"
)

type rulesetRule struct {
	Type       string          `json:"type"`
	Parameters json.RawMessage `json:"parameters,omitempty"`
}

type pullRequestParameters struct {
	RequiredApprovingReviewCount   int      `json:"required_approving_review_count"`
	RequireCodeOwnerReview         bool     `json:"require_code_owner_review"`
	RequireLastPushApproval        bool     `json:"require_last_push_approval"`
	DismissStaleReviewsOnPush      bool     `json:"dismiss_stale_reviews_on_push"`
	RequiredReviewThreadResolution bool     `json:"required_review_thread_resolution"`
	AllowedMergeMethods            []string `json:"allowed_merge_methods,omitempty"`
}

type statusCheckParameters struct {
	StrictRequiredStatusChecksPolicy bool                `json:"strict_required_status_checks_policy"`
	RequiredStatusChecks             []statusCheckTarget `json:"required_status_checks"`
}

type statusCheckTarget struct {
	Context string `json:"context"`
}

type refNameCondition struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

type rulesetConditions struct {
	RefName refNameCondition `json:"ref_name"`
}

type rulesetResponse struct {
	ID           int                `json:"id"`
	Name         string             `json:"name"`
	Target       string             `json:"target"`
	Enforcement  string             `json:"enforcement"`
	Conditions   *rulesetConditions `json:"conditions"`
	Rules        []rulesetRule      `json:"rules"`
	BypassActors []json.RawMessage  `json:"bypass_actors"`
}

// ListRulesets returns the repo's rulesets. GitHub omits each ruleset's rules
// from this summary, so a caller needing rules must fetch by ID.
func (c *Client) ListRulesets(owner, repo string) ([]Ruleset, error) {
	var response []rulesetResponse
	if err := c.Get(fmt.Sprintf("repos/%s/%s/rulesets", owner, repo), &response); err != nil {
		return nil, fmt.Errorf("failed to list rulesets: %w", err)
	}

	rulesets := make([]Ruleset, 0, len(response))
	for i := range response {
		rulesets = append(rulesets, response[i].toRuleset())
	}

	return rulesets, nil
}

// GetRuleset returns one ruleset with its rules populated.
func (c *Client) GetRuleset(owner, repo string, id int) (*Ruleset, error) {
	var response rulesetResponse
	if err := c.Get(fmt.Sprintf("repos/%s/%s/rulesets/%d", owner, repo, id), &response); err != nil {
		return nil, fmt.Errorf("failed to get ruleset %d: %w", id, err)
	}

	ruleset := response.toRuleset()

	return &ruleset, nil
}

// FindRulesetByName returns the named ruleset with its rules, or
// ErrRulesetNotFound. Names are unique per repo in GitHub's UI but not
// enforced by the API, so the first match wins.
func (c *Client) FindRulesetByName(owner, repo, name string) (*Ruleset, error) {
	rulesets, err := c.ListRulesets(owner, repo)
	if err != nil {
		return nil, err
	}

	for i := range rulesets {
		if rulesets[i].Name == name {
			return c.GetRuleset(owner, repo, rulesets[i].ID)
		}
	}

	return nil, fmt.Errorf("%w: %s/%s %q", ErrRulesetNotFound, owner, repo, name)
}

// CreateRuleset creates a new ruleset on the repo.
func (c *Client) CreateRuleset(owner, repo string, desired Ruleset) error {
	var response rulesetResponse
	if err := c.Post(fmt.Sprintf("repos/%s/%s/rulesets", owner, repo), desired.toRequest(), &response); err != nil {
		return fmt.Errorf("failed to create ruleset %q: %w", desired.Name, err)
	}

	return nil
}

// UpdateRuleset replaces an existing ruleset. Like branch protection, GitHub
// treats this as a full replacement, so desired must carry every rule to keep.
func (c *Client) UpdateRuleset(owner, repo string, id int, desired Ruleset) error {
	var response rulesetResponse
	path := fmt.Sprintf("repos/%s/%s/rulesets/%d", owner, repo, id)

	if err := c.Put(path, desired.toRequest(), &response); err != nil {
		return fmt.Errorf("failed to update ruleset %d: %w", id, err)
	}

	return nil
}

func (r rulesetResponse) toRuleset() Ruleset {
	ruleset := Ruleset{
		ID:           r.ID,
		Name:         r.Name,
		Target:       r.Target,
		Enforcement:  r.Enforcement,
		BypassActors: r.BypassActors,
	}

	if r.Conditions != nil {
		ruleset.IncludeRefs = r.Conditions.RefName.Include
		ruleset.ExcludeRefs = r.Conditions.RefName.Exclude
	}

	for _, rule := range r.Rules {
		switch rule.Type {
		case ruleDeletion:
			ruleset.BlockDeletion = true
		case ruleNonFastForward:
			ruleset.BlockForcePush = true
		case ruleLinearHistory:
			ruleset.RequireLinearHistory = true
		case rulePullRequest:
			ruleset.PullRequest = decodePullRequest(rule.Parameters)
		case ruleStatusChecks:
			ruleset.RequiredStatusChecks = decodeStatusChecks(rule.Parameters)
		default:
			ruleset.Unmanaged = append(ruleset.Unmanaged, mustMarshal(rule))
		}
	}

	return ruleset
}

// decodePullRequest returns the zero rule on malformed parameters: the rule's
// presence is what requires a PR, so a decode failure must not look like absence.
func decodePullRequest(raw json.RawMessage) *PullRequestRule {
	params := pullRequestParameters{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return &PullRequestRule{}
		}
	}

	return &PullRequestRule{
		RequiredApprovals:              params.RequiredApprovingReviewCount,
		RequireCodeOwnerReview:         params.RequireCodeOwnerReview,
		RequireLastPushApproval:        params.RequireLastPushApproval,
		DismissStaleReviewsOnPush:      params.DismissStaleReviewsOnPush,
		RequiredReviewThreadResolution: params.RequiredReviewThreadResolution,
		AllowedMergeMethods:            params.AllowedMergeMethods,
	}
}

func decodeStatusChecks(raw json.RawMessage) []string {
	params := statusCheckParameters{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil
		}
	}

	contexts := make([]string, 0, len(params.RequiredStatusChecks))
	for _, check := range params.RequiredStatusChecks {
		contexts = append(contexts, check.Context)
	}

	return contexts
}

func (r Ruleset) toRequest() map[string]any {
	rules := make([]rulesetRule, 0, len(r.Unmanaged)+modeledRuleTypes)

	if r.BlockDeletion {
		rules = append(rules, rulesetRule{Type: ruleDeletion})
	}
	if r.BlockForcePush {
		rules = append(rules, rulesetRule{Type: ruleNonFastForward})
	}
	if r.RequireLinearHistory {
		rules = append(rules, rulesetRule{Type: ruleLinearHistory})
	}
	if r.PullRequest != nil {
		rules = append(rules, rulesetRule{Type: rulePullRequest, Parameters: mustMarshal(pullRequestParameters{
			RequiredApprovingReviewCount:   r.PullRequest.RequiredApprovals,
			RequireCodeOwnerReview:         r.PullRequest.RequireCodeOwnerReview,
			RequireLastPushApproval:        r.PullRequest.RequireLastPushApproval,
			DismissStaleReviewsOnPush:      r.PullRequest.DismissStaleReviewsOnPush,
			RequiredReviewThreadResolution: r.PullRequest.RequiredReviewThreadResolution,
			AllowedMergeMethods:            r.PullRequest.AllowedMergeMethods,
		})})
	}
	if len(r.RequiredStatusChecks) > 0 {
		targets := make([]statusCheckTarget, 0, len(r.RequiredStatusChecks))
		for _, context := range r.RequiredStatusChecks {
			targets = append(targets, statusCheckTarget{Context: context})
		}
		rules = append(rules, rulesetRule{Type: ruleStatusChecks, Parameters: mustMarshal(statusCheckParameters{
			RequiredStatusChecks: targets,
		})})
	}

	for _, raw := range r.Unmanaged {
		var rule rulesetRule
		if err := json.Unmarshal(raw, &rule); err == nil {
			rules = append(rules, rule)
		}
	}

	target := r.Target
	if target == "" {
		target = rulesetTargetBranch
	}

	include := r.IncludeRefs
	if len(include) == 0 {
		include = []string{rulesetDefaultBranch}
	}

	exclude := r.ExcludeRefs
	if exclude == nil {
		exclude = []string{}
	}

	body := map[string]any{
		"name":        r.Name,
		"target":      target,
		"enforcement": r.Enforcement,
		"conditions":  rulesetConditions{RefName: refNameCondition{Include: include, Exclude: exclude}},
		"rules":       rules,
	}

	if r.BypassActors != nil {
		body["bypass_actors"] = r.BypassActors
	}

	return body
}

func mustMarshal(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage("null")
	}

	return raw
}
