#!/usr/bin/env -S uv run --quiet --script
# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "plotext>=5.3.2",
# ]
# ///
"""Analyze GitHub Actions workflow performance over time.

Usage:
    ./scripts/gha_perf.py                    # Analyze default workflows
    ./scripts/gha_perf.py --workflow api.yml # Analyze specific workflow
    ./scripts/gha_perf.py --limit 50         # Analyze more runs
    ./scripts/gha_perf.py --branch main      # Filter by branch
    ./scripts/gha_perf.py --csv output.csv   # Export to CSV
    ./scripts/gha_perf.py --job "Build Python Services"  # Filter specific job
    ./scripts/gha_perf.py --compare main     # Compare current branch vs main
    ./scripts/gha_perf.py --by-branch        # Group and compare all branches
    ./scripts/gha_perf.py --cache-only       # Use cached data only (no fetch)
"""

from __future__ import annotations

import argparse
import csv
import json
import subprocess
import sys
from dataclasses import asdict, dataclass
from datetime import datetime
from pathlib import Path
from textwrap import dedent
from typing import TYPE_CHECKING, Any

import plotext as plt

if TYPE_CHECKING:
    from collections.abc import Sequence

DEFAULT_WORKFLOWS = [
    "docker-build.yml",
    "api.yml",
    "tasker.yml",
    "common.yml",
    "orchestrator.yml",
]

DEFAULT_CACHE_PATH = Path(".gha-perf-cache.json")


@dataclass(frozen=True)
class StepTiming:
    name: str
    duration_seconds: float


@dataclass(frozen=True)
class JobTiming:
    name: str
    duration_seconds: float
    steps: list[StepTiming]


@dataclass(frozen=True)
class RunTiming:
    run_id: int
    workflow: str
    branch: str
    conclusion: str
    created_at: datetime
    duration_seconds: float
    jobs: list[JobTiming]


# =============================================================================
# Data Collection Layer
# =============================================================================


def _run_gh_command(args: list[str]) -> str:
    result = subprocess.run(
        ["gh", *args],
        capture_output=True,
        text=True,
        check=True,
    )
    return result.stdout


def _run_gh_api(endpoint: str, params: dict[str, str] | None = None) -> dict | list:
    args = ["gh", "api", endpoint]
    if params:
        for key, value in params.items():
            args.extend(["-f", f"{key}={value}"])

    result = subprocess.run(args, capture_output=True, text=True, check=True)
    return json.loads(result.stdout)


def _get_repo_info() -> str:
    result = subprocess.run(
        ["gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner"],
        capture_output=True,
        text=True,
        check=True,
    )
    return result.stdout.strip()


def _parse_datetime(dt_str: str) -> datetime:
    return datetime.fromisoformat(dt_str.replace("Z", "+00:00"))


def _calculate_duration(start: str, end: str) -> float:
    start_dt = _parse_datetime(start)
    end_dt = _parse_datetime(end)
    return (end_dt - start_dt).total_seconds()


def _fetch_workflow_runs(
    workflow: str,
    limit: int = 30,
    branch: str | None = None,
) -> list[dict]:
    repo = _get_repo_info()
    endpoint = f"/repos/{repo}/actions/workflows/{workflow}/runs"

    params: dict[str, str] = {
        "per_page": str(min(limit, 100)),
        "status": "completed",
    }
    if branch:
        params["branch"] = branch

    try:
        data = _run_gh_api(endpoint, params)
        if isinstance(data, dict):
            runs = data.get("workflow_runs", [])
        else:
            runs = data
        return [
            {
                "databaseId": r["id"],
                "conclusion": r["conclusion"],
                "createdAt": r["created_at"],
                "updatedAt": r["updated_at"],
                "headBranch": r["head_branch"],
                "status": r["status"],
            }
            for r in runs[:limit]
            if r.get("conclusion")
        ]
    except subprocess.CalledProcessError:
        args = [
            "run",
            "list",
            f"--workflow={workflow}",
            f"--limit={limit}",
            "--json",
            "databaseId,conclusion,createdAt,updatedAt,headBranch,status",
        ]
        if branch:
            args.append(f"--branch={branch}")
        output = _run_gh_command(args)
        runs = json.loads(output)
        return [r for r in runs if r["status"] == "completed" and r["conclusion"]]


def _fetch_run_details(run_id: int) -> dict:
    repo = _get_repo_info()

    try:
        run_data = _run_gh_api(f"/repos/{repo}/actions/runs/{run_id}")
        jobs_data = _run_gh_api(f"/repos/{repo}/actions/runs/{run_id}/jobs")

        if not isinstance(run_data, dict) or not isinstance(jobs_data, dict):
            raise ValueError("Unexpected API response format")

        jobs = []
        for job in jobs_data.get("jobs", []):
            steps = []
            for step in job.get("steps", []):
                if step.get("status") == "completed" and step.get("started_at") and step.get("completed_at"):
                    steps.append({
                        "name": step["name"],
                        "number": step["number"],
                        "status": step["status"],
                        "conclusion": step.get("conclusion"),
                        "startedAt": step["started_at"],
                        "completedAt": step["completed_at"],
                    })
            jobs.append({
                "name": job["name"],
                "status": job["status"],
                "conclusion": job.get("conclusion"),
                "startedAt": job.get("started_at"),
                "completedAt": job.get("completed_at"),
                "steps": steps,
            })

        return {
            "createdAt": run_data["created_at"],
            "updatedAt": run_data["updated_at"],
            "jobs": jobs,
        }
    except (subprocess.CalledProcessError, ValueError, KeyError):
        output = _run_gh_command([
            "run",
            "view",
            str(run_id),
            "--json",
            "jobs,createdAt,updatedAt",
        ])
        return json.loads(output)


def _parse_run_timing(
    run_id: int,
    workflow: str,
    branch: str,
    conclusion: str,
    details: dict,
) -> RunTiming:
    created_at = _parse_datetime(details["createdAt"])
    run_duration = _calculate_duration(details["createdAt"], details["updatedAt"])

    jobs = []
    for job in details.get("jobs", []):
        if job["status"] != "completed":
            continue
        job_duration = _calculate_duration(job["startedAt"], job["completedAt"])

        steps = []
        for step in job.get("steps", []):
            if step["status"] != "completed":
                continue
            step_duration = _calculate_duration(step["startedAt"], step["completedAt"])
            steps.append(StepTiming(name=step["name"], duration_seconds=step_duration))

        jobs.append(JobTiming(
            name=job["name"],
            duration_seconds=job_duration,
            steps=steps,
        ))

    return RunTiming(
        run_id=run_id,
        workflow=workflow,
        branch=branch,
        conclusion=conclusion,
        created_at=created_at,
        duration_seconds=run_duration,
        jobs=jobs,
    )


def fetch_timings(
    workflows: Sequence[str],
    limit: int,
    branch: str | None,
    existing_run_ids: set[int] | None = None,
) -> list[RunTiming]:
    existing_run_ids = existing_run_ids or set()
    timings: list[RunTiming] = []

    for workflow in workflows:
        print(f"Fetching runs for {workflow}...", file=sys.stderr)
        try:
            runs = _fetch_workflow_runs(workflow, limit=limit, branch=branch)
        except subprocess.CalledProcessError as err:
            print(f"  Warning: Could not fetch {workflow}: {err.stderr}", file=sys.stderr)
            continue

        new_runs = [r for r in runs if r["databaseId"] not in existing_run_ids]
        if len(new_runs) < len(runs):
            print(f"  Skipping {len(runs) - len(new_runs)} cached runs", file=sys.stderr)

        for run in new_runs:
            run_id = run["databaseId"]
            try:
                details = _fetch_run_details(run_id)
                timing = _parse_run_timing(
                    run_id=run_id,
                    workflow=workflow,
                    branch=run["headBranch"],
                    conclusion=run["conclusion"],
                    details=details,
                )
                timings.append(timing)
            except subprocess.CalledProcessError as err:
                print(f"  Warning: Could not fetch run {run_id}: {err.stderr}", file=sys.stderr)

    return timings


# =============================================================================
# Cache Layer
# =============================================================================


def _timing_to_dict(timing: RunTiming) -> dict[str, Any]:
    return {
        "run_id": timing.run_id,
        "workflow": timing.workflow,
        "branch": timing.branch,
        "conclusion": timing.conclusion,
        "created_at": timing.created_at.isoformat(),
        "duration_seconds": timing.duration_seconds,
        "jobs": [
            {
                "name": job.name,
                "duration_seconds": job.duration_seconds,
                "steps": [asdict(step) for step in job.steps],
            }
            for job in timing.jobs
        ],
    }


def _dict_to_timing(data: dict[str, Any]) -> RunTiming:
    return RunTiming(
        run_id=data["run_id"],
        workflow=data["workflow"],
        branch=data["branch"],
        conclusion=data["conclusion"],
        created_at=datetime.fromisoformat(data["created_at"]),
        duration_seconds=data["duration_seconds"],
        jobs=[
            JobTiming(
                name=job["name"],
                duration_seconds=job["duration_seconds"],
                steps=[StepTiming(**step) for step in job["steps"]],
            )
            for job in data["jobs"]
        ],
    )


def load_cache(cache_path: Path) -> list[RunTiming]:
    if not cache_path.exists():
        return []
    try:
        data = json.loads(cache_path.read_text())
        return [_dict_to_timing(item) for item in data.get("runs", [])]
    except (json.JSONDecodeError, KeyError) as err:
        print(f"Warning: Could not load cache: {err}", file=sys.stderr)
        return []


def save_cache(cache_path: Path, timings: list[RunTiming]) -> None:
    data = {
        "updated_at": datetime.now().isoformat(),
        "runs": [_timing_to_dict(t) for t in timings],
    }
    cache_path.write_text(json.dumps(data, indent=2))
    print(f"Cache saved: {cache_path} ({len(timings)} runs)", file=sys.stderr)


def merge_timings(existing: list[RunTiming], new: list[RunTiming]) -> list[RunTiming]:
    by_id = {t.run_id: t for t in existing}
    for t in new:
        by_id[t.run_id] = t
    return sorted(by_id.values(), key=lambda t: t.created_at)


# =============================================================================
# Data Processing Layer
# =============================================================================


def filter_timings(
    timings: list[RunTiming],
    workflows: Sequence[str] | None = None,
    branch: str | None = None,
) -> list[RunTiming]:
    result = timings
    if workflows:
        workflow_set = set(workflows)
        result = [t for t in result if t.workflow in workflow_set]
    if branch:
        result = [t for t in result if t.branch == branch]
    return result


def _format_duration(seconds: float) -> str:
    if seconds < 60:
        return f"{seconds:.0f}s"
    minutes = seconds / 60
    if minutes < 60:
        return f"{minutes:.1f}m"
    hours = minutes / 60
    return f"{hours:.1f}h"


def _compute_stats(durations: list[float]) -> tuple[float, float, float]:
    avg = sum(durations) / len(durations)
    min_d = min(durations)
    max_d = max(durations)
    return avg, min_d, max_d


def _get_current_branch() -> str:
    result = subprocess.run(
        ["git", "rev-parse", "--abbrev-ref", "HEAD"],
        capture_output=True,
        text=True,
        check=True,
    )
    return result.stdout.strip()


# =============================================================================
# Output Layer - CSV
# =============================================================================


def export_csv(timings: list[RunTiming], output_path: Path) -> None:
    with output_path.open("w", newline="") as f:
        writer = csv.writer(f)
        writer.writerow([
            "run_id",
            "workflow",
            "branch",
            "conclusion",
            "created_at",
            "run_duration_s",
            "job_name",
            "job_duration_s",
            "step_name",
            "step_duration_s",
        ])

        for timing in timings:
            for job in timing.jobs:
                for step in job.steps:
                    writer.writerow([
                        timing.run_id,
                        timing.workflow,
                        timing.branch,
                        timing.conclusion,
                        timing.created_at.isoformat(),
                        timing.duration_seconds,
                        job.name,
                        job.duration_seconds,
                        step.name,
                        step.duration_seconds,
                    ])

    print(f"Exported to {output_path}", file=sys.stderr)


# =============================================================================
# Output Layer - Terminal Summaries
# =============================================================================


def print_summary(timings: list[RunTiming]) -> None:
    print("\n" + "=" * 60)
    print("WORKFLOW PERFORMANCE SUMMARY")
    print("=" * 60)

    workflow_stats: dict[str, list[float]] = {}
    for timing in timings:
        workflow_stats.setdefault(timing.workflow, []).append(timing.duration_seconds)

    for workflow, durations in sorted(workflow_stats.items()):
        avg, min_d, max_d = _compute_stats(durations)
        print(dedent(f"""\
            {workflow}:
              Runs: {len(durations)}
              Avg:  {_format_duration(avg)}
              Min:  {_format_duration(min_d)}
              Max:  {_format_duration(max_d)}
            """))


def print_job_summary(timings: list[RunTiming]) -> None:
    print("\n" + "=" * 60)
    print("JOB PERFORMANCE SUMMARY (Top 10 by avg duration)")
    print("=" * 60)

    job_stats: dict[str, list[float]] = {}
    for timing in timings:
        for job in timing.jobs:
            key = f"{timing.workflow}:{job.name}"
            job_stats.setdefault(key, []).append(job.duration_seconds)

    sorted_jobs = sorted(
        job_stats.items(),
        key=lambda x: sum(x[1]) / len(x[1]),
        reverse=True,
    )[:10]

    for job_key, durations in sorted_jobs:
        avg = sum(durations) / len(durations)
        print(f"  {job_key[:50]}: {_format_duration(avg)} avg ({len(durations)} runs)")


def print_by_branch(timings: list[RunTiming], base_branch: str = "main") -> None:
    print("\n" + "=" * 70)
    print("PERFORMANCE BY BRANCH")
    print("=" * 70)

    branch_workflow_stats: dict[str, dict[str, list[float]]] = {}
    for t in timings:
        branch_workflow_stats.setdefault(t.branch, {}).setdefault(t.workflow, []).append(
            t.duration_seconds
        )

    all_workflows = sorted({wf for bdata in branch_workflow_stats.values() for wf in bdata})
    base_avgs: dict[str, float] = {}
    if base_branch in branch_workflow_stats:
        for wf, durs in branch_workflow_stats[base_branch].items():
            base_avgs[wf] = sum(durs) / len(durs)

    sorted_branches = sorted(
        branch_workflow_stats.keys(),
        key=lambda b: (b != base_branch, b),
    )

    for branch in sorted_branches:
        wf_stats = branch_workflow_stats[branch]
        total_runs = sum(len(durs) for durs in wf_stats.values())
        is_base = branch == base_branch

        print(f"\n{'[BASE] ' if is_base else ''}{branch} ({total_runs} runs)")
        print("-" * 50)

        for wf in all_workflows:
            if wf not in wf_stats:
                continue
            durs = wf_stats[wf]
            avg = sum(durs) / len(durs)

            delta_str = ""
            if not is_base and wf in base_avgs:
                diff = avg - base_avgs[wf]
                pct = (diff / base_avgs[wf]) * 100 if base_avgs[wf] else 0
                sign = "+" if diff > 0 else ""
                delta_str = f" ({sign}{pct:.0f}% vs {base_branch})"

            print(f"  {wf}: {_format_duration(avg)} avg ({len(durs)} runs){delta_str}")

    print("\n" + "-" * 70)
    print("JOB PERFORMANCE BY BRANCH (Top 10 slowest)")
    print("-" * 70)

    branch_job_stats: dict[str, dict[str, list[float]]] = {}
    for t in timings:
        branch_job_stats.setdefault(t.branch, {})
        for job in t.jobs:
            key = f"{t.workflow}:{job.name}"
            branch_job_stats[t.branch].setdefault(key, []).append(job.duration_seconds)

    base_job_avgs: dict[str, float] = {}
    if base_branch in branch_job_stats:
        for job_key, durs in branch_job_stats[base_branch].items():
            base_job_avgs[job_key] = sum(durs) / len(durs)

    all_job_avgs: list[tuple[str, str, float, float | None]] = []
    for branch, job_stats in branch_job_stats.items():
        for job_key, durs in job_stats.items():
            avg = sum(durs) / len(durs)
            base_avg = base_job_avgs.get(job_key) if branch != base_branch else None
            all_job_avgs.append((branch, job_key, avg, base_avg))

    sorted_jobs = sorted(all_job_avgs, key=lambda x: x[2], reverse=True)[:15]

    for branch, job_key, avg, base_avg in sorted_jobs:
        delta_str = ""
        if base_avg is not None:
            diff = avg - base_avg
            pct = (diff / base_avg) * 100 if base_avg else 0
            sign = "+" if diff > 0 else ""
            delta_str = f" ({sign}{pct:.0f}%)"

        branch_label = f"[{branch[:20]}]"
        print(f"  {branch_label:22} {job_key[:35]}: {_format_duration(avg)}{delta_str}")


def print_comparison(
    timings_a: list[RunTiming],
    timings_b: list[RunTiming],
    label_a: str,
    label_b: str,
) -> None:
    print("\n" + "=" * 70)
    print(f"BRANCH COMPARISON: {label_a} vs {label_b}")
    print("=" * 70)

    workflows_a: dict[str, list[float]] = {}
    workflows_b: dict[str, list[float]] = {}

    for t in timings_a:
        workflows_a.setdefault(t.workflow, []).append(t.duration_seconds)
    for t in timings_b:
        workflows_b.setdefault(t.workflow, []).append(t.duration_seconds)

    all_workflows = sorted(set(workflows_a.keys()) | set(workflows_b.keys()))

    for workflow in all_workflows:
        durs_a = workflows_a.get(workflow, [])
        durs_b = workflows_b.get(workflow, [])

        print(f"\n{workflow}:")
        if durs_a and durs_b:
            avg_a, _, _ = _compute_stats(durs_a)
            avg_b, _, _ = _compute_stats(durs_b)
            diff = avg_a - avg_b
            pct = (diff / avg_b) * 100 if avg_b else 0

            indicator = "FASTER" if diff < 0 else "SLOWER" if diff > 0 else "SAME"
            sign = "+" if diff > 0 else ""

            print(f"  {label_a}: {_format_duration(avg_a)} avg ({len(durs_a)} runs)")
            print(f"  {label_b}: {_format_duration(avg_b)} avg ({len(durs_b)} runs)")
            print(f"  Delta: {sign}{_format_duration(abs(diff))} ({sign}{pct:.1f}%) - {indicator}")
        elif durs_a:
            avg_a, _, _ = _compute_stats(durs_a)
            print(f"  {label_a}: {_format_duration(avg_a)} avg ({len(durs_a)} runs)")
            print(f"  {label_b}: No data")
        else:
            avg_b, _, _ = _compute_stats(durs_b)
            print(f"  {label_a}: No data")
            print(f"  {label_b}: {_format_duration(avg_b)} avg ({len(durs_b)} runs)")

    print("\n" + "-" * 70)
    print("JOB-LEVEL COMPARISON (Top changes)")
    print("-" * 70)

    jobs_a: dict[str, list[float]] = {}
    jobs_b: dict[str, list[float]] = {}

    for t in timings_a:
        for job in t.jobs:
            key = f"{t.workflow}:{job.name}"
            jobs_a.setdefault(key, []).append(job.duration_seconds)
    for t in timings_b:
        for job in t.jobs:
            key = f"{t.workflow}:{job.name}"
            jobs_b.setdefault(key, []).append(job.duration_seconds)

    all_jobs = set(jobs_a.keys()) & set(jobs_b.keys())
    job_diffs: list[tuple[str, float, float, float]] = []

    for job_key in all_jobs:
        avg_a = sum(jobs_a[job_key]) / len(jobs_a[job_key])
        avg_b = sum(jobs_b[job_key]) / len(jobs_b[job_key])
        diff = avg_a - avg_b
        job_diffs.append((job_key, avg_a, avg_b, diff))

    sorted_diffs = sorted(job_diffs, key=lambda x: abs(x[3]), reverse=True)[:10]

    for job_key, avg_a, avg_b, diff in sorted_diffs:
        pct = (diff / avg_b) * 100 if avg_b else 0
        sign = "+" if diff > 0 else ""
        indicator = "slower" if diff > 0 else "faster"
        print(f"  {job_key[:45]}")
        print(f"    {_format_duration(avg_a)} vs {_format_duration(avg_b)} ({sign}{pct:.0f}% {indicator})")


# =============================================================================
# Output Layer - Plots
# =============================================================================


def plot_workflow_totals(timings: list[RunTiming]) -> None:
    workflow_data: dict[str, list[tuple[datetime, float]]] = {}
    for timing in timings:
        workflow_data.setdefault(timing.workflow, []).append(
            (timing.created_at, timing.duration_seconds / 60)
        )

    plt.clear_figure()
    plt.title("Workflow Total Duration Over Time")
    plt.xlabel("Run Index")
    plt.ylabel("Duration (minutes)")

    for workflow, data in workflow_data.items():
        if not data:
            continue
        durations = [d[1] for d in data]
        plt.plot(list(range(len(durations))), durations, label=workflow.replace(".yml", ""))

    plt.theme("pro")
    plt.show()


def plot_job_breakdown(timings: list[RunTiming], workflow_filter: str | None = None) -> None:
    job_durations: dict[str, list[float]] = {}

    for timing in timings:
        if workflow_filter and timing.workflow != workflow_filter:
            continue
        for job in timing.jobs:
            job_durations.setdefault(job.name, []).append(job.duration_seconds / 60)

    if not job_durations:
        print("No job data to display", file=sys.stderr)
        return

    avg_durations = {
        name: sum(durs) / len(durs)
        for name, durs in job_durations.items()
        if durs
    }
    sorted_jobs = sorted(avg_durations.items(), key=lambda x: -x[1])[:10]

    plt.clear_figure()
    title = "Average Job Duration"
    if workflow_filter:
        title += f" ({workflow_filter})"
    plt.title(title)

    names = [j[0][:30] for j in sorted_jobs]
    values = [j[1] for j in sorted_jobs]

    plt.bar(names, values)
    plt.ylabel("Duration (minutes)")
    plt.theme("pro")
    plt.show()


def plot_step_breakdown(
    timings: list[RunTiming],
    job_filter: str | None = None,
) -> None:
    step_durations: dict[str, list[float]] = {}

    for timing in timings:
        for job in timing.jobs:
            if job_filter and job.name != job_filter:
                continue
            for step in job.steps:
                step_durations.setdefault(step.name, []).append(step.duration_seconds)

    if not step_durations:
        print("No step data to display", file=sys.stderr)
        return

    avg_durations = {
        name: sum(durs) / len(durs)
        for name, durs in step_durations.items()
        if durs
    }
    sorted_steps = sorted(avg_durations.items(), key=lambda x: -x[1])[:15]

    plt.clear_figure()
    title = "Average Step Duration"
    if job_filter:
        title += f" ({job_filter})"
    plt.title(title)

    names = [s[0][:40] for s in sorted_steps]
    values = [s[1] for s in sorted_steps]

    plt.bar(names, values)
    plt.ylabel("Duration (seconds)")
    plt.theme("pro")
    plt.show()


def plot_branch_comparison(
    timings_a: list[RunTiming],
    timings_b: list[RunTiming],
    label_a: str,
    label_b: str,
) -> None:
    workflows_a: dict[str, list[float]] = {}
    workflows_b: dict[str, list[float]] = {}

    for t in timings_a:
        workflows_a.setdefault(t.workflow, []).append(t.duration_seconds / 60)
    for t in timings_b:
        workflows_b.setdefault(t.workflow, []).append(t.duration_seconds / 60)

    all_workflows = sorted(set(workflows_a.keys()) | set(workflows_b.keys()))

    names = []
    vals_a = []
    vals_b = []

    for wf in all_workflows:
        short_name = wf.replace(".yml", "")
        names.append(short_name)
        vals_a.append(sum(workflows_a.get(wf, [0])) / max(len(workflows_a.get(wf, [1])), 1))
        vals_b.append(sum(workflows_b.get(wf, [0])) / max(len(workflows_b.get(wf, [1])), 1))

    plt.clear_figure()
    plt.title(f"Workflow Duration: {label_a} vs {label_b}")
    plt.multiple_bar(names, [vals_a, vals_b], label=[label_a[:20], label_b[:20]])
    plt.ylabel("Duration (minutes)")
    plt.theme("pro")
    plt.show()


def plot_by_branch(timings: list[RunTiming], base_branch: str = "main") -> None:
    branch_stats: dict[str, list[float]] = {}
    for t in timings:
        branch_stats.setdefault(t.branch, []).append(t.duration_seconds / 60)

    sorted_branches = sorted(
        branch_stats.keys(),
        key=lambda b: (b != base_branch, sum(branch_stats[b]) / len(branch_stats[b])),
    )[:10]

    names = []
    avgs = []
    for branch in sorted_branches:
        durs = branch_stats[branch]
        short_name = branch[:25] + ("..." if len(branch) > 25 else "")
        if branch == base_branch:
            short_name = f"* {short_name}"
        names.append(short_name)
        avgs.append(sum(durs) / len(durs))

    plt.clear_figure()
    plt.title("Average Workflow Duration by Branch")
    plt.bar(names, avgs)
    plt.ylabel("Duration (minutes)")
    plt.theme("pro")
    plt.show()


# =============================================================================
# Main Entry Point
# =============================================================================


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Analyze GitHub Actions workflow performance",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=dedent("""\
            Examples:
              %(prog)s                              # Default workflows
              %(prog)s -w docker-build.yml          # Single workflow
              %(prog)s -w api.yml -w tasker.yml     # Multiple workflows
              %(prog)s --branch main --limit 50    # Filter by branch
              %(prog)s --csv perf.csv              # Export data
              %(prog)s --job "Build Python Services" # Drill into job steps
              %(prog)s --compare main              # Compare current branch vs main
              %(prog)s --by-branch                 # Group and compare all branches
              %(prog)s --cache-only                # Use cached data only
            """),
    )
    parser.add_argument(
        "-w", "--workflow",
        action="append",
        dest="workflows",
        help="Workflow file to analyze (can specify multiple)",
    )
    parser.add_argument(
        "-l", "--limit",
        type=int,
        default=20,
        help="Number of runs to fetch per workflow (default: 20)",
    )
    parser.add_argument(
        "-b", "--branch",
        help="Filter by branch name",
    )
    parser.add_argument(
        "-c", "--compare",
        metavar="BRANCH",
        help="Compare current branch against another branch (e.g., main)",
    )
    parser.add_argument(
        "--csv",
        type=Path,
        help="Export detailed data to CSV file",
    )
    parser.add_argument(
        "-j", "--job",
        help="Show step breakdown for specific job name",
    )
    parser.add_argument(
        "--by-branch",
        action="store_true",
        help="Group runs by branch and compare against main",
    )
    parser.add_argument(
        "--base-branch",
        default="main",
        help="Base branch for comparisons (default: main)",
    )
    parser.add_argument(
        "--cache",
        type=Path,
        default=DEFAULT_CACHE_PATH,
        help=f"Cache file path (default: {DEFAULT_CACHE_PATH})",
    )
    parser.add_argument(
        "--cache-only",
        action="store_true",
        help="Use cached data only, do not fetch new runs",
    )
    parser.add_argument(
        "--no-cache",
        action="store_true",
        help="Do not use or update the cache",
    )
    parser.add_argument(
        "--no-plots",
        action="store_true",
        help="Skip terminal plots (useful for CI or CSV export only)",
    )

    args = parser.parse_args()
    workflows = args.workflows or DEFAULT_WORKFLOWS

    cached_timings = load_cache(args.cache) if not args.no_cache else []
    cached_run_ids = {t.run_id for t in cached_timings}

    if args.cache_only:
        all_timings = cached_timings
        print(f"Using {len(all_timings)} cached runs", file=sys.stderr)
    else:
        if args.compare:
            current_branch = _get_current_branch()
            print(f"Comparing {current_branch} vs {args.compare}...", file=sys.stderr)

            new_current = fetch_timings(workflows, args.limit, current_branch, cached_run_ids)
            new_base = fetch_timings(workflows, args.limit, args.compare, cached_run_ids)
            new_timings = new_current + new_base
        elif args.by_branch:
            new_timings = fetch_timings(workflows, args.limit, branch=None, existing_run_ids=cached_run_ids)
        else:
            new_timings = fetch_timings(workflows, args.limit, args.branch, cached_run_ids)

        all_timings = merge_timings(cached_timings, new_timings)

        if not args.no_cache and new_timings:
            save_cache(args.cache, all_timings)

        print(f"\nTotal: {len(all_timings)} runs ({len(new_timings)} new)", file=sys.stderr)

    if not all_timings:
        print("No runs found", file=sys.stderr)
        sys.exit(1)

    if args.csv:
        filtered = filter_timings(all_timings, workflows, args.branch)
        export_csv(filtered, args.csv)

    if args.compare:
        current_branch = _get_current_branch()
        timings_current = filter_timings(all_timings, workflows, current_branch)
        timings_base = filter_timings(all_timings, workflows, args.compare)

        print_comparison(timings_current, timings_base, current_branch, args.compare)

        if not args.no_plots and timings_current and timings_base:
            print("\n[Press Enter to see plots, Ctrl+C to exit]", file=sys.stderr)
            try:
                input()
            except KeyboardInterrupt:
                print()
                return

            plot_branch_comparison(timings_current, timings_base, current_branch, args.compare)
        return

    if args.by_branch:
        filtered = filter_timings(all_timings, workflows)
        print_by_branch(filtered, base_branch=args.base_branch)

        if not args.no_plots:
            print("\n[Press Enter to see plots, Ctrl+C to exit]", file=sys.stderr)
            try:
                input()
            except KeyboardInterrupt:
                print()
                return

            plot_by_branch(filtered, base_branch=args.base_branch)
        return

    filtered = filter_timings(all_timings, workflows, args.branch)

    print_summary(filtered)
    print_job_summary(filtered)

    if not args.no_plots:
        print("\n[Press Enter to see plots, Ctrl+C to exit]", file=sys.stderr)
        try:
            input()
        except KeyboardInterrupt:
            print()
            return

        plot_workflow_totals(filtered)

        if len(workflows) == 1:
            plot_job_breakdown(filtered, workflow_filter=workflows[0])

        if args.job:
            plot_step_breakdown(filtered, job_filter=args.job)
        else:
            plot_job_breakdown(filtered)


if __name__ == "__main__":
    main()
