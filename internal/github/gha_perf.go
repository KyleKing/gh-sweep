package github

import (
	"fmt"
	"sort"
	"time"
)

// StepTiming records the timing and outcome of a single job step.
type StepTiming struct {
	Name            string        `json:"name"`
	DurationSeconds float64       `json:"duration_seconds"`
	Status          string        `json:"status"`
	Conclusion      string        `json:"conclusion"`
	StartedAt       time.Time     `json:"started_at"`
	CompletedAt     time.Time     `json:"completed_at"`
	Duration        time.Duration `json:"-"`
}

// JobTiming records the timing and outcome of a single workflow job.
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

// RunTiming records the timing and outcome of a single workflow run.
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

// WorkflowFile identifies a workflow definition file in a repository.
type WorkflowFile struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Path  string `json:"path"`
	State string `json:"state"`
}

// WorkflowStats aggregates run timing and outcomes for one workflow.
type WorkflowStats struct {
	Workflow     string
	TotalRuns    int
	AvgDuration  time.Duration
	MinDuration  time.Duration
	MaxDuration  time.Duration
	SuccessRate  float64
	FailureCount int
}

// JobStats aggregates timing for one workflow job across runs.
type JobStats struct {
	WorkflowJob string
	TotalRuns   int
	AvgDuration time.Duration
	MinDuration time.Duration
	MaxDuration time.Duration
}

// BranchStats aggregates run timing for one branch, including its delta against a base branch.
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
		ID         int       `json:"id"`
		Name       string    `json:"name"`
		WorkflowID int       `json:"workflow_id"`
		Status     string    `json:"status"`
		Conclusion string    `json:"conclusion"`
		HeadBranch string    `json:"head_branch"`
		HeadSHA    string    `json:"head_sha"`
		CreatedAt  time.Time `json:"created_at"`
		UpdatedAt  time.Time `json:"updated_at"`
		Path       string    `json:"path"`
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

// ListWorkflows lists the workflow files defined in a repository.
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

// FetchWorkflowRunsOptions configures FetchWorkflowRuns.
type FetchWorkflowRunsOptions struct {
	WorkflowFile string
	Branch       string
	Status       string
	Limit        int
	CreatedAfter time.Time
}

const (
	defaultRunsLimit = 30
	maxRunsPerFetch  = 100
)

// FetchWorkflowRuns fetches completed workflow runs matching opts.
func (c *Client) FetchWorkflowRuns(
	owner, repo string,
	opts FetchWorkflowRunsOptions,
) ([]RunTiming, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultRunsLimit
	}
	if limit > maxRunsPerFetch {
		limit = maxRunsPerFetch
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
	for i := range response.WorkflowRuns {
		r := &response.WorkflowRuns[i]
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

// FetchRunDetails fetches job and step timing for a single workflow run.
func (c *Client) FetchRunDetails(owner, repo string, runID int) (*RunTiming, error) {
	path := fmt.Sprintf("repos/%s/%s/actions/runs/%d/jobs", owner, repo, runID)

	var response jobsResponse
	if err := c.Get(path, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch run details: %w", err)
	}

	var jobs []JobTiming
	for i := range response.Jobs {
		j := &response.Jobs[i]
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

// FetchWorkflowRunsWithDetails fetches workflow runs and enriches each with job timing details.
func (c *Client) FetchWorkflowRunsWithDetails(
	owner, repo string,
	opts FetchWorkflowRunsOptions,
) ([]RunTiming, error) {
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

// ComputeWorkflowStats aggregates run timing and success rate per workflow.
func ComputeWorkflowStats(runs []RunTiming) map[string]*WorkflowStats {
	stats := make(map[string]*WorkflowStats)

	for i := range runs {
		r := &runs[i]
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
		case ConclusionSuccess:
		case ConclusionFailure:
			s.FailureCount++
		}
	}

	for _, s := range stats {
		if s.TotalRuns > 0 {
			s.AvgDuration /= time.Duration(s.TotalRuns)
			successCount := s.TotalRuns - s.FailureCount
			s.SuccessRate = float64(successCount) / float64(s.TotalRuns) * percentMultiplier
		}
	}

	return stats
}

// ComputeJobStats aggregates job timing per workflow:job key.
func ComputeJobStats(runs []RunTiming) map[string]*JobStats {
	stats := make(map[string]*JobStats)

	for i := range runs {
		r := &runs[i]
		for k := range r.Jobs {
			j := &r.Jobs[k]
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
			s.AvgDuration /= time.Duration(s.TotalRuns)
		}
	}

	return stats
}

// accumulateBranchRun folds one run's duration into its branch and
// branch:workflow buckets, creating either bucket on first sight.
func accumulateBranchRun(stats map[string]*BranchStats, r *RunTiming) {
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

// finalizeBranchAverages converts each bucket's summed duration into an average.
func finalizeBranchAverages(stats map[string]*BranchStats) {
	for _, s := range stats {
		if s.TotalRuns > 0 {
			s.AvgDuration /= time.Duration(s.TotalRuns)
		}
		for _, ws := range s.WorkflowStats {
			if ws.TotalRuns > 0 {
				ws.AvgDuration /= time.Duration(ws.TotalRuns)
			}
		}
	}
}

// applyBranchDeltas sets each non-base branch's average-duration delta against baseBranch.
func applyBranchDeltas(stats map[string]*BranchStats, baseBranch string) {
	baseStats, ok := stats[baseBranch]
	if !ok || baseStats.AvgDuration <= 0 {
		return
	}

	for branch, s := range stats {
		if branch == baseBranch {
			continue
		}

		delta := s.AvgDuration - baseStats.AvgDuration
		s.DeltaVsBase = float64(delta) / float64(time.Second)
		s.DeltaVsBasePct = float64(delta) / float64(baseStats.AvgDuration) * percentMultiplier
	}
}

// ComputeBranchStats aggregates run timing per branch, including each non-base
// branch's average-duration delta against baseBranch.
func ComputeBranchStats(runs []RunTiming, baseBranch string) map[string]*BranchStats {
	stats := make(map[string]*BranchStats)

	for i := range runs {
		accumulateBranchRun(stats, &runs[i])
	}

	finalizeBranchAverages(stats)
	applyBranchDeltas(stats, baseBranch)

	return stats
}

// FilterRunsByBranch returns the runs matching branch, or all runs when branch is empty.
func FilterRunsByBranch(runs []RunTiming, branch string) []RunTiming {
	if branch == "" {
		return runs
	}
	var filtered []RunTiming
	for i := range runs {
		if runs[i].Branch == branch {
			filtered = append(filtered, runs[i])
		}
	}

	return filtered
}

// FilterRunsByWorkflows returns the runs whose workflow is in workflows, or all runs when workflows is empty.
func FilterRunsByWorkflows(runs []RunTiming, workflows []string) []RunTiming {
	if len(workflows) == 0 {
		return runs
	}
	workflowSet := make(map[string]bool)
	for _, w := range workflows {
		workflowSet[w] = true
	}
	var filtered []RunTiming
	for i := range runs {
		if workflowSet[runs[i].Workflow] {
			filtered = append(filtered, runs[i])
		}
	}

	return filtered
}

// FilterRunsByTimeRange returns the runs created within [since, until], treating a zero bound as unbounded.
func FilterRunsByTimeRange(runs []RunTiming, since, until time.Time) []RunTiming {
	var filtered []RunTiming
	for i := range runs {
		if !since.IsZero() && runs[i].CreatedAt.Before(since) {
			continue
		}
		if !until.IsZero() && runs[i].CreatedAt.After(until) {
			continue
		}
		filtered = append(filtered, runs[i])
	}

	return filtered
}

// SortRunsByDate sorts runs in place by creation time.
func SortRunsByDate(runs []RunTiming, ascending bool) {
	sort.Slice(runs, func(i, j int) bool {
		if ascending {
			return runs[i].CreatedAt.Before(runs[j].CreatedAt)
		}

		return runs[i].CreatedAt.After(runs[j].CreatedAt)
	})
}

// GetTopJobsByDuration returns the jobs with the highest average duration, capped at limit (0 means unlimited).
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

// FormatDuration renders a duration as seconds, minutes, or hours depending on its magnitude.
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}

	return fmt.Sprintf("%.1fh", d.Hours())
}
