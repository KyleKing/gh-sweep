package models

import "time"

// Repository represents a GitHub repository reference
type Repository struct {
	Owner string
	Name  string
}

// FullName returns the full repository name (owner/name)
func (r Repository) FullName() string {
	return r.Owner + "/" + r.Name
}

// ParseRepository parses a repository string into owner and name
func ParseRepository(repo string) (*Repository, error) {
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

	if len(parts) != 2 {
		return nil, nil // Return nil for invalid format
	}

	return &Repository{
		Owner: parts[0],
		Name:  parts[1],
	}, nil
}

// BranchNode represents a node in the branch tree
type BranchNode struct {
	Name           string
	SHA            string
	Ahead          int
	Behind         int
	LastCommitDate time.Time
	Parent         *BranchNode
	Children       []*BranchNode
}

// AddChild adds a child node to this branch
func (n *BranchNode) AddChild(child *BranchNode) {
	child.Parent = n
	n.Children = append(n.Children, child)
}

// IsLeaf returns true if this node has no children
func (n *BranchNode) IsLeaf() bool {
	return len(n.Children) == 0
}

// Depth returns the depth of this node in the tree
func (n *BranchNode) Depth() int {
	depth := 0
	current := n.Parent
	for current != nil {
		depth++
		current = current.Parent
	}
	return depth
}
