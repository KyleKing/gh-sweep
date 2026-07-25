package github

import (
	"errors"
	"testing"
)

func TestDeleteBlocked(t *testing.T) {
	tests := []struct {
		name    string
		branch  BranchStatus
		wantErr error
	}{
		{
			name:    "plain branch is deletable",
			branch:  BranchStatus{Branch: Branch{Name: "feature"}},
			wantErr: nil,
		},
		{
			name:    "default branch is blocked",
			branch:  BranchStatus{Branch: Branch{Name: "main"}, IsDefault: true},
			wantErr: ErrDefaultBranchDeletion,
		},
		{
			name:    "protected branch is blocked",
			branch:  BranchStatus{Branch: Branch{Name: "release", Protected: true}},
			wantErr: ErrProtectedBranchDeletion,
		},
		{
			name: "branch with open PR is blocked",
			branch: BranchStatus{
				Branch: Branch{Name: "feature"},
				PR:     &PullRequest{Number: 7, State: "open"},
			},
			wantErr: ErrOpenPRBranchDeletion,
		},
		{
			name: "branch with closed PR is deletable",
			branch: BranchStatus{
				Branch: Branch{Name: "feature"},
				PR:     &PullRequest{Number: 7, State: "closed"},
			},
			wantErr: nil,
		},
		{
			name: "default branch blocked before open PR",
			branch: BranchStatus{
				Branch:    Branch{Name: "main"},
				IsDefault: true,
				PR:        &PullRequest{Number: 3, State: "open"},
			},
			wantErr: ErrDefaultBranchDeletion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.branch.DeleteBlocked()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("DeleteBlocked() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestMatchBranchPR(t *testing.T) {
	repoFullName := "owner/repo"
	prs := []PullRequest{
		{Number: 1, State: "closed", Head: PRRef{Ref: "feature", Repo: repoFullName}},
		{Number: 2, State: "closed", Head: PRRef{Ref: "feature", Repo: repoFullName}},
		{Number: 3, State: "open", Head: PRRef{Ref: "active", Repo: repoFullName}},
		{Number: 4, State: "closed", Head: PRRef{Ref: "forked", Repo: "other/fork"}},
		{Number: 5, State: "closed", Head: PRRef{Ref: "legacy", Repo: ""}},
	}

	tests := []struct {
		name       string
		branch     string
		wantNumber int
	}{
		{name: "no matching PR", branch: "orphan", wantNumber: 0},
		{name: "open PR wins", branch: "active", wantNumber: 3},
		{name: "latest closed PR wins", branch: "feature", wantNumber: 2},
		{name: "fork head is ignored", branch: "forked", wantNumber: 0},
		{name: "missing head repo still matches", branch: "legacy", wantNumber: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := MatchBranchPR(prs, repoFullName, tt.branch)

			switch {
			case tt.wantNumber == 0 && pr != nil:
				t.Errorf("MatchBranchPR() = #%d, want nil", pr.Number)
			case tt.wantNumber != 0 && pr == nil:
				t.Errorf("MatchBranchPR() = nil, want #%d", tt.wantNumber)
			case pr != nil && pr.Number != tt.wantNumber:
				t.Errorf("MatchBranchPR() = #%d, want #%d", pr.Number, tt.wantNumber)
			}
		})
	}
}
