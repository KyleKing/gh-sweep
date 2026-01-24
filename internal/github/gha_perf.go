package github

import (
	"fmt"
	"sort"
	"time"
)

type StepTiming struct {
	Name            string        `json:"name"`
	DurationSeconds float64       `json:"duration_seconds"`
	Status          string        `json:"status"`
	Conclusion      string        `json:"conclusion"`
	StartedAt       time.Time     `json:"started_at"`
	CompletedAt     time.Time     `json:"completed_at"`
	Duration        time.Duration `json:"-"`
}

type JobTiming struct {
	Name            string        `json:"name"`
	DurationSeconds float64       `json:"duration_seconds"`
	Status          string        `json:"status"`
	Conclusion      string        `json:"conclusion"`
	StartedAt       time.Time     `json:"started_at"`
	CompletedAt     time.Time     `json:"completed_at"`
	Duration        time.Duration `json:"-"`
	Steps           []StepTiming  `json:"steps"`
}

type RunTiming struct {
	RunID           int           `json:"run_id"`
	Workflow        string        `json:"workflow"`
	WorkflowID      int           `json:"workflow_id"`
	Branch          string        `json:"branch"`
	HeadSHA         string        `json:"head_sha"`
	Conclusion      string        `json:"conclusion"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	DurationSeconds float64       `json:"duration_seconds"`
	Duration        time.Duration `json:"-"`
	Jobs            []JobTiming   `json:"jobs"`
}

type WorkflowFile struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Path  string `json:"path"`
	State string `json:"state"`
}

type WorkflowStats struct {
	Workflow     string
	TotalRuns    int
	AvgDuration  time.Duration
	MinDuration  time.Duration
	MaxDuration  time.Duration
	SuccessRate  float64
	FailureCount int
}

type JobStats struct {
	WorkflowJob string
	TotalRuns   int
	AvgDuration time.Duration
	MinDuration time.Duration
	MaxDuration time.Duration
}

type BranchStats struct {
	Branch         string
	TotalRuns      int
	AvgDuration    time.Duration
	WorkflowStats  map[string]*WorkflowStats
	DeltaVsBase    float64
	DeltaVsBasePct float64
}

type workflowsResponse struct {
	Workflows []struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Path  string `json:"path"`
		State string `json:"state"`
	} `json:"workflows"`
}

type workflowRunsDetailResponse struct {
	WorkflowRuns []struct {
		ID           int       `json:"id"`
		Name         string    `json:"name"`
		WorkflowID   int       `json:"workflow_id"`
		Status       string    `json:"status"`
		Conclusion   string    `json:"conclusion"`
		HeadBranch   string    `json:"head_branch"`
		HeadSHA      string    `json:"head_sha"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		Path         string    `json:"path"`
	} `json:"workflow_runs"`
}

type jobsResponse struct {
	Jobs []struct {
		ID          int       `json:"id"`
		Name        string    `json:"name"`
		Status      string    `json:"status"`
		Conclusion  string    `json:"conclusion"`
		StartedAt   time.Time `json:"started_at"`
		CompletedAt time.Time `json:"completed_at"`
		Steps       []struct {
			Name        string    `json:"name"`
			Number      int       `json:"number"`
			Status      string    `json:"status"`
			Conclusion  string    `json:"conclusion"`
			StartedAt   time.Time `json:"started_at"`
			CompletedAt time.Time `json:"completed_at"`
		} `json:"steps"`
	} `json:"jobs"`
}

func (c *Client) ListWorkflows(owner, repo string) ([]WorkflowFile, error) {
	var response workflowsResponse
	path := fmt.Sprintf("repos/%s/%s/actions/workflows", owner, repo)

	if err := c.Get(path, &response); err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}

	workflows := make([]WorkflowFile, len(response.Workflows))
	for i, w := range response.Workflows {
		workflows[i] = WorkflowFile{
			ID:    w.ID,
			Name:  w.Name,
			Path:  w.Path,
			State: w.State,
		}
	}

	return workflows, nil
}

type FetchWorkflowRunsOptions struct {
	WorkflowFile string
	Branch       string
	Status       string
	Limit        int
	CreatedAfter time.Time
}

func (c *Client) FetchWorkflowRuns(owner, repo string, opts FetchWorkflowRunsOptions) ([]RunTiming, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}

	var path string
	if opts.WorkflowFile != "" {
		path = fmt.Sprintf("repos/%s/%s/actions/workflows/%s/runs?per_page=%d&status=completed",
			owner, repo, opts.WorkflowFile, limit)
	} else {
		path = fmt.Sprintf("repos/%s/%s/actions/runs?per_page=%d&status=completed",
			owner, repo, limit)
	}

	if opts.Branch != "" {
		path += "&branch=" + opts.Branch
	}

	var response workflowRunsDetailResponse
	if err := c.Get(path, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch workflow runs: %w", err)
	}

	var runs []RunTiming
	for _, r := range response.WorkflowRuns {
		if r.Conclusion == "" {
			continue
		}
		if !opts.CreatedAfter.IsZero() && r.CreatedAt.Before(opts.CreatedAfter) {
			continue
		}

		workflowName := r.Path
		if workflowName == "" {
			workflowName = r.Name
		}

		duration := r.UpdatedAt.Sub(r.CreatedAt)
		runs = append(runs, RunTiming{
			RunID:           r.ID,
			Workflow:        workflowName,
			WorkflowID:      r.WorkflowID,
			Branch:          r.HeadBranch,
			HeadSHA:         r.HeadSHA,
			Conclusion:      r.Conclusion,
			CreatedAt:       r.CreatedAt,
			UpdatedAt:       r.UpdatedAt,
			DurationSeconds: duration.Seconds(),
			Duration:        duration,
		})
	}

	return runs, nil
}

func (c *Client) FetchRunDetails(owner, repo string, runID int) (*RunTiming, error) {
	path := fmt.Sprintf("repos/%s/%s/actions/runs/%d/jobs", owner, repo, runID)

	var response jobsResponse
	if err := c.Get(path, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch run details: %w", err)
	}

	var jobs []JobTiming
	for _, j := range response.Jobs {
		if j.Status != "completed" {
			continue
		}

		var steps []StepTiming
		for _, s := range j.Steps {
			if s.Status != "completed" || s.StartedAt.IsZero() || s.CompletedAt.IsZero() {
				continue
			}

			duration := s.CompletedAt.Sub(s.StartedAt)
			steps = append(steps, StepTiming{
				Name:            s.Name,
				DurationSeconds: duration.Seconds(),
				Status:          s.Status,
				Conclusion:      s.Conclusion,
				StartedAt:       s.StartedAt,
				CompletedAt:     s.CompletedAt,
				Duration:        duration,
			})
		}

		jobDuration := j.CompletedAt.Sub(j.StartedAt)
		jobs = append(jobs, JobTiming{
			Name:            j.Name,
			DurationSeconds: jobDuration.Seconds(),
			Status:          j.Status,
			Conclusion:      j.Conclusion,
			StartedAt:       j.StartedAt,
			CompletedAt:     j.CompletedAt,
			Duration:        jobDuration,
			Steps:           steps,
		})
	}

	return &RunTiming{Jobs: jobs}, nil
}

func (c *Client) FetchWorkflowRunsWithDetails(owner, repo string, opts FetchWorkflowRunsOptions) ([]RunTiming, error) {
	runs, err := c.FetchWorkflowRuns(owner, repo, opts)
	if err != nil {
		return nil, err
	}

	for i := range runs {
		details, err := c.FetchRunDetails(owner, repo, runs[i].RunID)
		if err != nil {
			continue
		}
		runs[i].Jobs = details.Jobs
	}

	return runs, nil
}

func ComputeWorkflowStats(runs []RunTiming) map[string]*WorkflowStats {
	stats := make(map[string]*WorkflowStats)

	for _, r := range runs {
		wf := r.Workflow
		if _, ok := stats[wf]; !ok {
			stats[wf] = &WorkflowStats{
				Workflow:    wf,
				MinDuration: r.Duration,
				MaxDuration: r.Duration,
			}
		}

		s := stats[wf]
		s.TotalRuns++
		s.AvgDuration += r.Duration

		if r.Duration < s.MinDuration {
			s.MinDuration = r.Duration
		}
		if r.Duration > s.MaxDuration {
			s.MaxDuration = r.Duration
		}

		switch r.Conclusion {
		case "success":
		case "failure":
			s.FailureCount++
		}
	}

	for _, s := range stats {
		if s.TotalRuns > 0 {
			s.AvgDuration = s.AvgDuration / time.Duration(s.TotalRuns)
			successCount := s.TotalRuns - s.FailureCount
			s.SuccessRate = float64(successCount) / float64(s.TotalRuns) * 100
		}
	}

	return stats
}

func ComputeJobStats(runs []RunTiming) map[string]*JobStats {
	stats := make(map[string]*JobStats)

	for _, r := range runs {
		for _, j := range r.Jobs {
			key := fmt.Sprintf("%s:%s", r.Workflow, j.Name)
			if _, ok := stats[key]; !ok {
				stats[key] = &JobStats{
					WorkflowJob: key,
					MinDuration: j.Duration,
					MaxDuration: j.Duration,
				}
			}

			s := stats[key]
			s.TotalRuns++
			s.AvgDuration += j.Duration

			if j.Duration < s.MinDuration {
				s.MinDuration = j.Duration
			}
			if j.Duration > s.MaxDuration {
				s.MaxDuration = j.Duration
			}
		}
	}

	for _, s := range stats {
		if s.TotalRuns > 0 {
			s.AvgDuration = s.AvgDuration / time.Duration(s.TotalRuns)
		}
	}

	return stats
}

func ComputeBranchStats(runs []RunTiming, baseBranch string) map[string]*BranchStats {
	stats := make(map[string]*BranchStats)

	for _, r := range runs {
		branch := r.Branch
		if _, ok := stats[branch]; !ok {
			stats[branch] = &BranchStats{
				Branch:        branch,
				WorkflowStats: make(map[string]*WorkflowStats),
			}
		}

		s := stats[branch]
		s.TotalRuns++
		s.AvgDuration += r.Duration

		wf := r.Workflow
		if _, ok := s.WorkflowStats[wf]; !ok {
			s.WorkflowStats[wf] = &WorkflowStats{
				Workflow:    wf,
				MinDuration: r.Duration,
				MaxDuration: r.Duration,
			}
		}

		ws := s.WorkflowStats[wf]
		ws.TotalRuns++
		ws.AvgDuration += r.Duration
		if r.Duration < ws.MinDuration {
			ws.MinDuration = r.Duration
		}
		if r.Duration > ws.MaxDuration {
			ws.MaxDuration = r.Duration
		}
	}

	for _, s := range stats {
		if s.TotalRuns > 0 {
			s.AvgDuration = s.AvgDuration / time.Duration(s.TotalRuns)
		}
		for _, ws := range s.WorkflowStats {
			if ws.TotalRuns > 0 {
				ws.AvgDuration = ws.AvgDuration / time.Duration(ws.TotalRuns)
			}
		}
	}

	if baseStats, ok := stats[baseBranch]; ok {
		for branch, s := range stats {
			if branch == baseBranch {
				continue
			}
			if baseStats.AvgDuration > 0 {
				s.DeltaVsBase = float64(s.AvgDuration-baseStats.AvgDuration) / float64(time.Second)
				s.DeltaVsBasePct = float64(s.AvgDuration-baseStats.AvgDuration) / float64(baseStats.AvgDuration) * 100
			}
		}
	}

	return stats
}

func FilterRunsByBranch(runs []RunTiming, branch string) []RunTiming {
	if branch == "" {
		return runs
	}
	var filtered []RunTiming
	for _, r := range runs {
		if r.Branch == branch {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func FilterRunsByWorkflows(runs []RunTiming, workflows []string) []RunTiming {
	if len(workflows) == 0 {
		return runs
	}
	workflowSet := make(map[string]bool)
	for _, w := range workflows {
		workflowSet[w] = true
	}
	var filtered []RunTiming
	for _, r := range runs {
		if workflowSet[r.Workflow] {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func FilterRunsByTimeRange(runs []RunTiming, since, until time.Time) []RunTiming {
	var filtered []RunTiming
	for _, r := range runs {
		if !since.IsZero() && r.CreatedAt.Before(since) {
			continue
		}
		if !until.IsZero() && r.CreatedAt.After(until) {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

func SortRunsByDate(runs []RunTiming, ascending bool) {
	sort.Slice(runs, func(i, j int) bool {
		if ascending {
			return runs[i].CreatedAt.Before(runs[j].CreatedAt)
		}
		return runs[i].CreatedAt.After(runs[j].CreatedAt)
	})
}

func GetTopJobsByDuration(stats map[string]*JobStats, limit int) []*JobStats {
	var jobs []*JobStats
	for _, s := range stats {
		jobs = append(jobs, s)
	}

	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].AvgDuration > jobs[j].AvgDuration
	})

	if limit > 0 && len(jobs) > limit {
		jobs = jobs[:limit]
	}

	return jobs
}

func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}
