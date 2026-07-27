package github

import (
	"fmt"
	"time"
)

const viewerRepoWatchInfoQuery = `query($cursor: String) {
  viewer {
    login
    repositories(first: 100, after: $cursor, ownerAffiliations: [OWNER], orderBy: {field: NAME, direction: ASC}) {
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        name
        nameWithOwner
        owner { login }
        isPrivate
        isArchived
        isFork
        viewerSubscription
        viewerCanSubscribe
        stargazerCount
        pushedAt
        updatedAt
        watchers { totalCount }
      }
    }
  }
}`

// RepoWatchInfo is a repo's watch state plus metadata GitHub's REST
// subscription endpoint doesn't expose (activity, popularity, archival),
// fetched in a single paginated GraphQL query rather than one REST call per repo.
//
// GitHub's "Custom" per-notification-type watch setting has no representation
// in either the REST or GraphQL API: a repo set to Custom on github.com reports
// the same viewerSubscription as one left at the default (see
// https://github.com/orgs/community/discussions/65099). State should be read
// as "the best this API can tell us," not as ground truth for Custom repos.
type RepoWatchInfo struct {
	RepoBasic
	IsArchived         bool
	IsFork             bool
	State              WatchState
	ViewerCanSubscribe bool
	StargazerCount     int
	WatcherCount       int
	PushedAt           time.Time
	UpdatedAt          time.Time
}

type viewerRepoWatchInfoResponse struct {
	Viewer struct {
		Login        string `json:"login"`
		Repositories struct {
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
			Nodes []repoWatchNode `json:"nodes"`
		} `json:"repositories"`
	} `json:"viewer"`
}

type repoWatchNode struct {
	Name          string `json:"name"`
	NameWithOwner string `json:"nameWithOwner"`
	Owner         struct {
		Login string `json:"login"`
	} `json:"owner"`
	IsPrivate          bool      `json:"isPrivate"`
	IsArchived         bool      `json:"isArchived"`
	IsFork             bool      `json:"isFork"`
	ViewerSubscription string    `json:"viewerSubscription"`
	ViewerCanSubscribe bool      `json:"viewerCanSubscribe"`
	StargazerCount     int       `json:"stargazerCount"`
	PushedAt           time.Time `json:"pushedAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
	Watchers           struct {
		TotalCount int `json:"totalCount"`
	} `json:"watchers"`
}

func watchStateFromViewerSubscription(s string) WatchState {
	switch s {
	case "SUBSCRIBED":
		return WatchStateSubscribed
	case "IGNORED":
		return WatchStateIgnored
	default:
		return WatchStateDefault
	}
}

// ListViewerRepoWatchInfo fetches watch state and enrichment metadata for every
// repository owned by the authenticated user, paginated via GraphQL. Unlike the
// REST subscription endpoint, the query is atomic per page: a page either
// returns full data for every repo in it or fails outright, so there's no
// partial-failure state to silently misreport as "not watching".
func (g *GQLClient) ListViewerRepoWatchInfo() (string, []RepoWatchInfo, error) {
	var username string
	var infos []RepoWatchInfo
	var cursor interface{}

	for {
		variables := map[string]interface{}{"cursor": cursor}

		var response viewerRepoWatchInfoResponse
		if err := g.doer.Do(viewerRepoWatchInfoQuery, variables, &response); err != nil {
			return "", nil, fmt.Errorf("failed to list repo watch info: %w", err)
		}

		username = response.Viewer.Login

		for _, node := range response.Viewer.Repositories.Nodes {
			infos = append(infos, RepoWatchInfo{
				RepoBasic: RepoBasic{
					Name:     node.Name,
					FullName: node.NameWithOwner,
					Owner:    node.Owner.Login,
					Private:  node.IsPrivate,
				},
				IsArchived:         node.IsArchived,
				IsFork:             node.IsFork,
				State:              watchStateFromViewerSubscription(node.ViewerSubscription),
				ViewerCanSubscribe: node.ViewerCanSubscribe,
				StargazerCount:     node.StargazerCount,
				WatcherCount:       node.Watchers.TotalCount,
				PushedAt:           node.PushedAt,
				UpdatedAt:          node.UpdatedAt,
			})
		}

		if !response.Viewer.Repositories.PageInfo.HasNextPage {
			break
		}
		cursor = response.Viewer.Repositories.PageInfo.EndCursor
	}

	return username, infos, nil
}
