package models

import (
	"testing"
)

func TestRepositoryFullName(t *testing.T) {
	t.Parallel()

	r := Repository{Owner: "acme", Name: "widgets"}
	if got := r.FullName(); got != "acme/widgets" {
		t.Errorf("FullName() = %q, want acme/widgets", got)
	}
}

func TestParseRepository(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantOwner string
		wantName  string
		wantNil   bool
	}{
		{"valid", "acme/widgets", "acme", "widgets", false},
		{"missing slash", "acme", "", "", true},
		{"too many parts", "a/b/c", "", "", true},
		{"empty", "", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := ParseRepository(tt.input)

			if tt.wantNil {
				if repo != nil {
					t.Fatalf("ParseRepository(%q) = %+v, want nil", tt.input, repo)
				}

				return
			}

			if repo == nil {
				t.Fatalf("ParseRepository(%q) = nil, want repository", tt.input)
			}

			if repo.Owner != tt.wantOwner || repo.Name != tt.wantName {
				t.Errorf("ParseRepository(%q) = %s/%s, want %s/%s",
					tt.input, repo.Owner, repo.Name, tt.wantOwner, tt.wantName)
			}
		})
	}
}

func TestBranchNodeTree(t *testing.T) {
	t.Parallel()

	root := &BranchNode{Name: "main"}
	child := &BranchNode{Name: "feature/login"}
	grandchild := &BranchNode{Name: "feature/login-fix"}

	root.AddChild(child)
	child.AddChild(grandchild)

	if child.Parent != root {
		t.Error("expected child.Parent to be root")
	}

	if root.IsLeaf() {
		t.Error("root with children reported as leaf")
	}

	if !grandchild.IsLeaf() {
		t.Error("grandchild without children reported as non-leaf")
	}

	tests := []struct {
		node *BranchNode
		want int
	}{
		{root, 0},
		{child, 1},
		{grandchild, 2},
	}

	for _, tt := range tests {
		if got := tt.node.Depth(); got != tt.want {
			t.Errorf("Depth(%s) = %d, want %d", tt.node.Name, got, tt.want)
		}
	}
}
