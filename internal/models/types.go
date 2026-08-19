// Package models holds the domain types shared across gh-sweep's CLI and
// TUI: repository references and the branch tree used to render ancestry.
package models

import "time"

const repositoryPartsCount = 2

// Repository represents a GitHub repository reference.
type Repository struct {
	Owner string
	Name  string
}

// FullName returns the full repository name (owner/name).
func (r Repository) FullName() string {
	return r.Owner + "/" + r.Name
}

// ParseRepository parses a repository string into owner and name, or returns
// nil when repo isn't in "owner/name" format.
func ParseRepository(repo string) *Repository {
	// Simple parsing, can be enhanced
	parts := []string{}
	current := ""
	for _, char := range repo {
		if char == '/' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	if len(parts) != repositoryPartsCount {
		return nil
	}

	return &Repository{
		Owner: parts[0],
		Name:  parts[1],
	}
}

// BranchNode represents a node in the branch tree.
type BranchNode struct {
	Name           string
	SHA            string
	Ahead          int
	Behind         int
	LastCommitDate time.Time
	Parent         *BranchNode
	Children       []*BranchNode
}

// AddChild adds a child node to this branch.
func (n *BranchNode) AddChild(child *BranchNode) {
	child.Parent = n
	n.Children = append(n.Children, child)
}

// IsLeaf returns true if this node has no children.
func (n *BranchNode) IsLeaf() bool {
	return len(n.Children) == 0
}

// Depth returns the depth of this node in the tree.
func (n *BranchNode) Depth() int {
	depth := 0
	current := n.Parent
	for current != nil {
		depth++
		current = current.Parent
	}

	return depth
}
