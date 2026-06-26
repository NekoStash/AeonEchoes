package indexing

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakePendingRunner struct {
	mu         sync.Mutex
	calls      int
	results    []RunResult
	errs       []error
	callCh     chan struct{}
	blockUntil <-chan struct{}
}

func (r *fakePendingRunner) RunPending(ctx context.Context, projectID string, limit int) (RunResult, error) {
	if projectID != "" {
		return RunResult{}, fmt.Errorf("unexpected projectID %q", projectID)
	}
	if limit <= 0 {
		return RunResult{}, fmt.Errorf("unexpected limit %d", limit)
	}
	if r.blockUntil != nil {
		select {
		case <-ctx.Done():
			return RunResult{}, ctx.Err()
		case <-r.blockUntil:
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	if r.callCh != nil {
		select {
		case r.callCh <- struct{}{}:
		default:
		}
	}
	var result RunResult
	var err error
	if len(r.results) > 0 {
		result = r.results[0]
		r.results = r.results[1:]
	}
	if len(r.errs) > 0 {
		err = r.errs[0]
		r.errs = r.errs[1:]
	}
	return result, err
}

func TestWorkerNotifyTriggersRun(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	runner := &fakePendingRunner{
		results: []RunResult{{Count: 1}, {Count: 1}},
		callCh:  make(chan struct{}, 4),
	}
	worker, err := NewWorker(runner, logger, time.Second, 3, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWorker() error: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	waitForWorkerCall(t, runner.callCh, "startup")

	worker.Notify()
	waitForWorkerCall(t, runner.callCh, "notify")
	cancel()
	waitForWorkerStop(t, done)
	if !strings.Contains(logs.String(), "trigger=notify") {
		t.Fatalf("expected notify trigger in logs, got %q", logs.String())
	}
}

func TestWorkerNotifyDebouncesBurst(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	runner := &fakePendingRunner{callCh: make(chan struct{}, 8)}
	worker, err := NewWorker(runner, logger, time.Second, 2, 25*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWorker() error: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	waitForWorkerCall(t, runner.callCh, "startup")

	worker.Notify()
	worker.Notify()
	worker.Notify()
	waitForWorkerCall(t, runner.callCh, "debounced notify")
	assertNoWorkerCall(t, runner.callCh, 120*time.Millisecond, "unexpected second debounced notify run")

	cancel()
	waitForWorkerStop(t, done)
	runner.mu.Lock()
	calls := runner.calls
	runner.mu.Unlock()
	if calls != 2 {
		t.Fatalf("worker calls = %d, want startup + one debounced notify", calls)
	}
}

func TestWorkerTickerFallbackStillRuns(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	runner := &fakePendingRunner{callCh: make(chan struct{}, 8)}
	worker, err := NewWorker(runner, logger, 30*time.Millisecond, 2, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWorker() error: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	waitForWorkerCall(t, runner.callCh, "startup")
	waitForWorkerCall(t, runner.callCh, "ticker")

	cancel()
	waitForWorkerStop(t, done)
}

func TestWorkerExitsAfterContextCancel(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	block := make(chan struct{})
	runner := &fakePendingRunner{blockUntil: block}
	worker, err := NewWorker(runner, logger, time.Second, 2, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWorker() error: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	time.Sleep(20 * time.Millisecond)
	cancel()
	close(block)
	waitForWorkerStop(t, done)
	if !strings.Contains(logs.String(), "index worker stopped") {
		t.Fatalf("expected worker stopped log, got %q", logs.String())
	}
}

func waitForWorkerCall(t *testing.T, callCh <-chan struct{}, label string) {
	t.Helper()
	select {
	case <-callCh:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("worker did not process %s in time", label)
	}
}

func assertNoWorkerCall(t *testing.T, callCh <-chan struct{}, wait time.Duration, label string) {
	t.Helper()
	select {
	case <-callCh:
		t.Fatal(label)
	case <-time.After(wait):
	}
}

func waitForWorkerStop(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Worker.Run() error: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("worker did not stop after cancel")
	}
}
