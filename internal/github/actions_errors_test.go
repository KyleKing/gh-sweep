package github

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestFetchFailedJobLogs(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/repos/acme/widgets/actions/runs/101/jobs":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Body: io.NopCloser(strings.NewReader(`{"jobs":[
					{"id":1,"name":"build","conclusion":"success"},
					{"id":2,"name":"test","conclusion":"failure","completed_at":"2026-01-10T12:00:00Z"}
				]}`)),
				Request: req,
			}, nil

		case "/repos/acme/widgets/actions/jobs/2/logs":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader("line one\r\nline two\r\n")),
				Request:    req,
			}, nil

		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader(`{"message":"Not Found"}`)),
				Request:    req,
			}, nil
		}
	})

	client, err := NewClientWithTransport(context.Background(), transport)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	logs, err := client.FetchFailedJobLogs("acme", "widgets", 101)
	if err != nil {
		t.Fatalf("FetchFailedJobLogs() error = %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("logs = %+v, want just the failed job", logs)
	}
	if logs[0].JobName != "test" || logs[0].JobID != 2 {
		t.Errorf("log = %+v", logs[0])
	}
	if len(logs[0].Lines) != 3 || logs[0].Lines[0] != "line one" || logs[0].Lines[1] != "line two" {
		t.Errorf("Lines = %v, want [line one, line two, \"\"]", logs[0].Lines)
	}
}

func TestFetchFailedJobLogsSkipsUnfetchableLogs(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/repos/acme/widgets/actions/runs/101/jobs":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader(`{"jobs":[{"id":1,"name":"test","conclusion":"failure"}]}`)),
				Request:    req,
			}, nil

		default:
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader("boom")),
				Request:    req,
			}, nil
		}
	})

	client, err := NewClientWithTransport(context.Background(), transport)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	logs, err := client.FetchFailedJobLogs("acme", "widgets", 101)
	if err != nil {
		t.Fatalf("FetchFailedJobLogs() error = %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("logs = %+v, want none (log fetch failure is skipped, not fatal)", logs)
	}
}
