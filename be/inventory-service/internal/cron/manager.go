package cron

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Manager schedules and runs registered cron Jobs with overlap protection,
// panic recovery, and graceful shutdown.
type Manager struct {
	mu      sync.Mutex
	jobs    []Job
	jobSeen map[string]struct{}
	started bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewManager creates a new cron Manager.
func NewManager() *Manager {
	return &Manager{
		jobSeen: make(map[string]struct{}),
	}
}

// Register adds one or more jobs to the manager before Start is called.
func (m *Manager) Register(jobs ...Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return fmt.Errorf("cannot register jobs after cron manager has started")
	}

	for _, j := range jobs {
		if j == nil {
			continue
		}
		name := j.Name()
		if name == "" {
			return fmt.Errorf("cron job name cannot be empty")
		}
		if j.Interval() <= 0 {
			return fmt.Errorf("cron job %q interval must be positive", name)
		}
		if _, exists := m.jobSeen[name]; exists {
			return fmt.Errorf("duplicate cron job name %q", name)
		}
		m.jobSeen[name] = struct{}{}
		m.jobs = append(m.jobs, j)
	}
	return nil
}

// Start launches a background worker goroutine for each registered job.
// It is safe to call Stop() to wait for all in-flight jobs to finish.
func (m *Manager) Start(ctx context.Context) {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.started = true
	jobs := make([]Job, len(m.jobs))
	copy(jobs, m.jobs)
	m.mu.Unlock()

	for _, j := range jobs {
		job := j
		m.wg.Add(1)
		go m.runLoop(runCtx, job)
		log.Printf("[CronManager] Registered job=%q interval=%s", job.Name(), job.Interval())
	}
}

// Stop cancels all scheduled job loops and blocks until in-flight executions complete.
func (m *Manager) Stop() {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return
	}
	cancel := m.cancel
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	m.wg.Wait()
	log.Printf("[CronManager] Stopped all cron jobs")
}

func (m *Manager) runLoop(ctx context.Context, job Job) {
	defer m.wg.Done()

	ticker := time.NewTicker(job.Interval())
	defer ticker.Stop()

	var running atomic.Bool

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !running.CompareAndSwap(false, true) {
				log.Printf("[CronManager] Skipping job=%q because previous execution is still running", job.Name())
				continue
			}

			m.executeOnce(ctx, job)
			running.Store(false)
		}
	}
}

func (m *Manager) executeOnce(ctx context.Context, job Job) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[CronManager] Recovered panic in job=%q: %v", job.Name(), r)
		}
	}()

	start := time.Now()
	if err := job.Run(ctx); err != nil {
		log.Printf("[CronManager] Job=%q failed after %s: %v", job.Name(), time.Since(start), err)
		return
	}
	log.Printf("[CronManager] Job=%q completed in %s", job.Name(), time.Since(start))
}
