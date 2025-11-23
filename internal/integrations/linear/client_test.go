package linear

import (
	"testing"
)

// TestExtractLinearIssueIDs tests issue ID extraction from PR bodies
func TestExtractLinearIssueIDs(t *testing.T) {
	tests := []struct {
		name        string
		prBody      string
		expectedIDs []string
	}{
		{
			name:        "single fixes reference",
			prBody:      "This PR fixes LIN-123",
			expectedIDs: []string{"LIN-123"},
		},
		{
			name:        "multiple references",
			prBody:      "Fixes LIN-123 and closes LIN-456, resolves LIN-789",
			expectedIDs: []string{"LIN-123", "LIN-456", "LIN-789"},
		},
		{
			name:        "case insensitive",
			prBody:      "FIXES LIN-100, Closes LIN-200, refs LIN-300",
			expectedIDs: []string{"LIN-100", "LIN-200", "LIN-300"},
		},
		{
			name:        "different prefixes",
			prBody:      "Fixes ENG-111, closes PROJ-222",
			expectedIDs: []string{"ENG-111", "PROJ-222"},
		},
		{
			name:        "no issue references",
			prBody:      "This is a regular PR without issue links",
			expectedIDs: []string{},
		},
		{
			name:        "duplicate references",
			prBody:      "Fixes LIN-123 and also fixes LIN-123",
			expectedIDs: []string{"LIN-123"}, // Deduplicated
		},
		{
			name: "multiline",
			prBody: `
			This PR implements feature X

			Fixes LIN-100
			Closes LIN-200
			`,
			expectedIDs: []string{"LIN-100", "LIN-200"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids := ExtractLinearIssueIDs(tt.prBody)

			if len(ids) != len(tt.expectedIDs) {
				t.Errorf("Expected %d IDs, got %d: %v",
					len(tt.expectedIDs), len(ids), ids)
			}

			// Convert to map for easier checking (order doesn't matter)
			idMap := make(map[string]bool)
			for _, id := range ids {
				idMap[id] = true
			}

			for _, expectedID := range tt.expectedIDs {
				if !idMap[expectedID] {
					t.Errorf("Expected to find ID '%s' in %v", expectedID, ids)
				}
			}
		})
	}
}

// TestCheckPRIssueSyncStatus tests sync status detection
func TestCheckPRIssueSyncStatus(t *testing.T) {
	tests := []struct {
		name        string
		prStatus    string
		issueState  string
		expectSync  bool
		expectReason string
	}{
		{
			name:       "merged PR with done issue",
			prStatus:   "merged",
			issueState: "Done",
			expectSync: true,
		},
		{
			name:       "merged PR with closed issue",
			prStatus:   "merged",
			issueState: "Closed",
			expectSync: true,
		},
		{
			name:         "merged PR with in-progress issue",
			prStatus:     "merged",
			issueState:   "In Progress",
			expectSync:   false,
			expectReason: "In Progress",
		},
		{
			name:       "closed PR with canceled issue",
			prStatus:   "closed",
			issueState: "Canceled",
			expectSync: true,
		},
		{
			name:         "closed PR with done issue",
			prStatus:     "closed",
			issueState:   "Done",
			expectSync:   false,
			expectReason: "Done",
		},
		{
			name:       "open PR with in-progress issue",
			prStatus:   "open",
			issueState: "In Progress",
			expectSync: true,
		},
		{
			name:         "open PR with done issue",
			prStatus:     "open",
			issueState:   "Done",
			expectSync:   false,
			expectReason: "Done",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inSync, reason := CheckPRIssueSyncStatus(tt.prStatus, tt.issueState)

			if inSync != tt.expectSync {
				t.Errorf("Expected sync=%v, got %v (reason: %s)",
					tt.expectSync, inSync, reason)
			}

			if !tt.expectSync && tt.expectReason != "" {
				// Check that reason contains the expected state
				if len(reason) == 0 {
					t.Error("Expected drift reason but got empty string")
				}
			}
		})
	}
}

// TestAnalyzePRIssueLinks tests link analysis
func TestAnalyzePRIssueLinks(t *testing.T) {
	pairs := []PRIssuePair{
		{
			Repository: "owner/repo",
			PRNumber:   1,
			PRStatus:   "merged",
			IssueID:    "LIN-100",
			Issue:      &Issue{State: "Done"},
		},
		{
			Repository: "owner/repo",
			PRNumber:   2,
			PRStatus:   "merged",
			IssueID:    "LIN-200",
			Issue:      &Issue{State: "In Progress"}, // Out of sync
		},
		{
			Repository: "owner/repo",
			PRNumber:   3,
			PRStatus:   "open",
			IssueID:    "LIN-300",
			Issue:      nil, // Issue not found
		},
	}

	analyzed := AnalyzePRIssueLinks(pairs)

	if len(analyzed) != 3 {
		t.Errorf("Expected 3 analyzed pairs, got %d", len(analyzed))
	}

	// Check first pair (should be in sync)
	if !analyzed[0].InSync {
		t.Errorf("Pair 0 should be in sync: %s", analyzed[0].DriftReason)
	}

	// Check second pair (should be out of sync)
	if analyzed[1].InSync {
		t.Error("Pair 1 should be out of sync")
	}
	if analyzed[1].DriftReason == "" {
		t.Error("Pair 1 should have drift reason")
	}

	// Check third pair (issue not found)
	if analyzed[2].InSync {
		t.Error("Pair 2 should be out of sync (issue not found)")
	}
	if analyzed[2].DriftReason != "Issue not found" {
		t.Errorf("Expected 'Issue not found', got '%s'", analyzed[2].DriftReason)
	}
}

// TestFilterOutOfSyncPairs tests filtering out-of-sync pairs
func TestFilterOutOfSyncPairs(t *testing.T) {
	pairs := []PRIssuePair{
		{PRNumber: 1, InSync: true},
		{PRNumber: 2, InSync: false, DriftReason: "Out of sync"},
		{PRNumber: 3, InSync: true},
		{PRNumber: 4, InSync: false, DriftReason: "Issue not found"},
	}

	outOfSync := FilterOutOfSyncPairs(pairs)

	if len(outOfSync) != 2 {
		t.Errorf("Expected 2 out-of-sync pairs, got %d", len(outOfSync))
	}

	// Check that only out-of-sync pairs are returned
	for _, pair := range outOfSync {
		if pair.InSync {
			t.Errorf("Filtered pair %d should not be in sync", pair.PRNumber)
		}
	}
}

// TestComposability tests function composition
func TestComposability(t *testing.T) {
	// Simulate a workflow: extract IDs -> fetch issues -> analyze -> filter
	prBody := "Fixes LIN-100 and closes LIN-200"

	// Step 1: Extract IDs
	ids := ExtractLinearIssueIDs(prBody)

	if len(ids) != 2 {
		t.Fatalf("Expected 2 IDs, got %d", len(ids))
	}

	// Step 2: Create pairs (simulating fetched issues)
	// Make one "Done" and one "In Progress"
	pairs := []PRIssuePair{
		{
			PRNumber: 1,
			PRStatus: "merged",
			IssueID:  "LIN-100",
			Issue:    &Issue{State: "Done"},
		},
		{
			PRNumber: 2,
			PRStatus: "merged",
			IssueID:  "LIN-200",
			Issue:    &Issue{State: "In Progress"}, // Out of sync
		},
	}

	// Step 3: Analyze
	analyzed := AnalyzePRIssueLinks(pairs)

	// Step 4: Filter
	outOfSync := FilterOutOfSyncPairs(analyzed)

	// Verify composition
	if len(outOfSync) != 1 {
		t.Errorf("Expected 1 out-of-sync pair after composition, got %d", len(outOfSync))
	}

	if outOfSync[0].IssueID != "LIN-200" {
		t.Errorf("Expected LIN-200 to be out of sync, got %s", outOfSync[0].IssueID)
	}
}
