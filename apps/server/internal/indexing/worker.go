package indexing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// PendingJobRunner is the minimal indexing surface required by the automatic worker.
type PendingJobRunner interface {
	RunPending(ctx context.Context, projectID string, limit int) (RunResult, error)
}

// WakeNotifier is the tiny surface used by HTTP handlers to wake the local index worker.
type WakeNotifier interface {
	Notify()
}

// Worker periodically advances pending indexing jobs in the background.
type Worker struct {
	runner    PendingJobRunner
	logger    *slog.Logger
	interval  time.Duration
	debounce  time.Duration
	batchSize int
	notifyCh  chan struct{}
}

func NewWorker(runner PendingJobRunner, logger *slog.Logger, interval time.Duration, batchSize int, debounce time.Duration) (*Worker, error) {
	if runner == nil {
		return nil, fmt.Errorf("index worker runner is not configured")
	}
	if logger == nil {
		return nil, fmt.Errorf("index worker logger is not configured")
	}
	if interval <= 0 {
		return nil, fmt.Errorf("index worker interval must be positive")
	}
	if batchSize <= 0 {
		return nil, fmt.Errorf("index worker batch size must be positive")
	}
	if debounce < 0 {
		return nil, fmt.Errorf("index worker wake debounce must not be negative")
	}
	return &Worker{runner: runner, logger: logger, interval: interval, debounce: debounce, batchSize: batchSize, notifyCh: make(chan struct{}, 1)}, nil
}

func (w *Worker) Notify() {
	if w == nil {
		return
	}
	select {
	case w.notifyCh <- struct{}{}:
	default:
	}
}

func (w *Worker) Run(ctx context.Context) error {
	if w == nil {
		return fmt.Errorf("index worker is not configured")
	}
	if ctx == nil {
		return fmt.Errorf("index worker context must not be nil")
	}
	w.logger.Info("index worker started", "interval", w.interval.String(), "batch_size", w.batchSize, "wake_debounce", w.debounce.String())
	defer w.logger.Info("index worker stopped")

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	var wakeTimer *time.Timer
	var wakeCh <-chan time.Time
	wakePending := false
	clearWake := func() {
		if wakeTimer != nil {
			if !wakeTimer.Stop() && wakeCh != nil {
				select {
				case <-wakeCh:
				default:
				}
			}
			wakeTimer = nil
		}
		wakeCh = nil
		wakePending = false
	}
	runTrigger := func(trigger string) error {
		clearWake()
		if err := w.runOnce(ctx, trigger); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
				return nil
			}
			w.logger.Error("index worker run pending failed", "trigger", trigger, "error", err)
		}
		return nil
	}

	if err := runTrigger("startup"); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			clearWake()
			return nil
		case <-ticker.C:
			if err := runTrigger("ticker"); err != nil {
				return err
			}
		case <-w.notifyCh:
			if w.debounce == 0 {
				if err := runTrigger("notify"); err != nil {
					return err
				}
				continue
			}
			if wakePending {
				continue
			}
			wakeTimer = time.NewTimer(w.debounce)
			wakeCh = wakeTimer.C
			wakePending = true
		case <-wakeCh:
			if err := runTrigger("notify"); err != nil {
				return err
			}
		}
	}
}

func (w *Worker) runOnce(ctx context.Context, trigger string) error {
	result, err := w.runner.RunPending(ctx, "", w.batchSize)
	if err != nil {
		w.logger.Error("index worker processing failed", "trigger", trigger, "error", err, "processed_count", result.Count)
		return err
	}
	if result.Count > 0 {
		w.logger.Info("index worker processed pending jobs", "trigger", trigger, "count", result.Count)
	}
	return nil
}
