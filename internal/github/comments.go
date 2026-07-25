package github

import (
	"fmt"
	"strings"
	"time"
)

// ReviewComment is a single comment within a PR review thread.
type ReviewComment struct {
	Author    string
	Body      string
	CreatedAt time.Time
	URL       string
}

// ReviewThread is a review conversation on a pull request.
type ReviewThread struct {
	Repository string
	PRNumber   int
	PRTitle    string
	Path       string
	IsResolved bool
	IsOutdated bool
	Comments   []ReviewComment
}

// FirstComment returns the thread's opening comment, if any.
func (t ReviewThread) FirstComment() (ReviewComment, bool) {
	if len(t.Comments) == 0 {
		return ReviewComment{}, false
	}

	return t.Comments[0], true
}

// LastActivity returns the creation time of the most recent comment.
func (t ReviewThread) LastActivity() time.Time {
	var latest time.Time
	for _, c := range t.Comments {
		if c.CreatedAt.After(latest) {
			latest = c.CreatedAt
		}
	}

	return latest
}

// FilterUnresolvedThreads keeps only threads that are not resolved.
func FilterUnresolvedThreads(threads []ReviewThread) []ReviewThread {
	unresolved := []ReviewThread{}
	for _, t := range threads {
		if !t.IsResolved {
			unresolved = append(unresolved, t)
		}
	}

	return unresolved
}

// ThreadFilter narrows review threads by author, activity date, and text.
type ThreadFilter struct {
	Author string
	Since  *time.Time
	Search string
}

// Apply returns the threads matching every set filter field.
func (f ThreadFilter) Apply(threads []ReviewThread) []ReviewThread {
	matched := []ReviewThread{}
	for _, t := range threads {
		if f.matches(t) {
			matched = append(matched, t)
		}
	}

	return matched
}

func (f ThreadFilter) matches(t ReviewThread) bool {
	if f.Author != "" && !threadHasAuthor(t, f.Author) {
		return false
	}
	if f.Since != nil && t.LastActivity().Before(*f.Since) {
		return false
	}
	if f.Search != "" && !threadContains(t, f.Search) {
		return false
	}

	return true
}

func threadHasAuthor(t ReviewThread, author string) bool {
	for _, c := range t.Comments {
		if strings.EqualFold(c.Author, author) {
			return true
		}
	}

	return false
}

func threadContains(t ReviewThread, search string) bool {
	needle := strings.ToLower(search)
	if strings.Contains(strings.ToLower(t.Path), needle) {
		return true
	}
	for _, c := range t.Comments {
		if strings.Contains(strings.ToLower(c.Body), needle) {
			return true
		}
	}

	return false
}

// ParseSinceDate parses a YYYY-MM-DD date for --since filtering.
func ParseSinceDate(value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid since date %q, expected YYYY-MM-DD: %w", value, err)
	}

	return parsed, nil
}
