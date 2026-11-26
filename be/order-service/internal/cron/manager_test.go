package cron

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type stubJob struct {
	name     string
	interval time.Duration
	runFn    func(ctx context.Context) error
}

func (s *stubJob) Name() string            { return s.name }
func (s *stubJob) Interval() time.Duration { return s.interval }
func (s *stubJob) Run(ctx context.Context) error {
	if s.runFn != nil {
		return s.runFn(ctx)
	}
	return nil
}

func TestManager_RegisterValidation(t *testing.T) {
	m := NewManager()

	if err := m.Register(&stubJob{name: "", interval: time.Second}); err == nil {
		t.Fatal("expected error for empty job name")
	}
	if err := m.Register(&stubJob{name: "bad-interval", interval: 0}); err == nil {
		t.Fatal("expected error for non-positive interval")
	}
	if err := m.Register(&stubJob{name: "job-a", interval: time.Second}); err != nil {
		t.Fatalf("unexpected error registering job-a: %v", err)
	}
	if err := m.Register(&stubJob{name: "job-a", interval: time.Second}); err == nil {
		t.Fatal("expected error for duplicate job name")
	}
}

func TestManager_RunsMultipleJobsAndRecoversPanic(t *testing.T) {
	m := NewManager()

	var job1Runs atomic.Int32
	var job2Runs atomic.Int32

	err := m.Register(
		&stubJob{
			name:     "job-1",
			interval: 15 * time.Millisecond,
			runFn: func(ctx context.Context) error {
				job1Runs.Add(1)
				return nil
			},
		},
		&stubJob{
			name:     "job-panic",
			interval: 15 * time.Millisecond,
			runFn: func(ctx context.Context) error {
				job2Runs.Add(1)
				panic("simulated panic")
			},
		},
	)
	if err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.Start(ctx)
	time.Sleep(60 * time.Millisecond)
	m.Stop()

	if job1Runs.Load() == 0 {
		t.Fatal("expected job-1 to run at least once")
	}
	if job2Runs.Load() == 0 {
		t.Fatal("expected job-panic to run at least once")
	}
}
