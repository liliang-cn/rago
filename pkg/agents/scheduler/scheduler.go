// Package scheduler implements task scheduling for the Agent pillar.
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"
	
	"github.com/liliang-cn/rago/v2/pkg/core"
)

// Scheduler manages scheduled tasks and jobs.
type Scheduler struct {
	mu     sync.RWMutex
	config Config
	jobs   map[string]*Job
	
	// Event handling
	eventHandlers map[string][]EventHandler
	eventChan     chan Event
	
	// Runtime state
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// Config holds scheduler configuration.
type Config struct {
	MaxConcurrentJobs  int
	DefaultRetryPolicy core.RetryPolicy
	PersistencePath    string
}

// Job represents a scheduled job.
type Job struct {
	ID          string
	Name        string
	Type        JobType
	Schedule    Schedule
	Handler     JobHandler
	Payload     map[string]interface{}
	RetryPolicy *core.RetryPolicy
	
	// Runtime state
	NextRun   time.Time
	LastRun   time.Time
	Status    JobStatus
	RunCount  int
	ErrorCount int
	LastError  error
	
	// Cron entry ID
	cronID int
}

// JobType defines the type of job.
type JobType string

const (
	JobTypeWorkflow JobType = "workflow"
	JobTypeAgent    JobType = "agent"
	JobTypeCustom   JobType = "custom"
)

// JobStatus represents the job status.
type JobStatus string

const (
	JobStatusScheduled JobStatus = "scheduled"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusDisabled  JobStatus = "disabled"
)

// Schedule defines when a job should run.
type Schedule struct {
	Type       ScheduleType
	Expression string
	Timezone   string
	Metadata   map[string]interface{}
}

// ScheduleType defines the type of schedule.
type ScheduleType string

const (
	ScheduleTypeCron     ScheduleType = "cron"
	ScheduleTypeInterval ScheduleType = "interval"
	ScheduleTypeEvent    ScheduleType = "event"
	ScheduleTypeOnce     ScheduleType = "once"
)

// JobHandler is a function that executes a job.
type JobHandler func(ctx context.Context, job *Job) error

// Event represents a system event that can trigger jobs.
type Event struct {
	Type      string
	Source    string
	Timestamp time.Time
	Data      map[string]interface{}
}

// EventHandler handles events.
type EventHandler func(event Event) error

// NewScheduler creates a new scheduler instance.
func NewScheduler(config Config) (*Scheduler, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	scheduler := &Scheduler{
		config:        config,
		jobs:          make(map[string]*Job),
		eventHandlers: make(map[string][]EventHandler),
		eventChan:     make(chan Event, 100),
		running:       false,
		ctx:           ctx,
		cancel:        cancel,
	}
	
	// Start the scheduler
	if err := scheduler.Start(); err != nil {
		return nil, err
	}
	
	return scheduler, nil
}

// Start starts the scheduler.
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.running {
		return fmt.Errorf("scheduler already running")
	}
	
	
	// Start event processor
	s.wg.Add(1)
	go s.processEvents()
	
	// Start job monitor
	s.wg.Add(1)
	go s.monitorJobs()
	
	s.running = true
	return nil
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.running {
		return fmt.Errorf("scheduler not running")
	}
	
	
	// Cancel context
	s.cancel()
	
	// Wait for goroutines
	s.wg.Wait()
	
	s.running = false
	return nil
}

// ScheduleJob schedules a new job.
func (s *Scheduler) ScheduleJob(job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.jobs[job.ID]; exists {
		return fmt.Errorf("job %s already exists", job.ID)
	}
	
	// Set default retry policy if not provided
	if job.RetryPolicy == nil {
		job.RetryPolicy = &s.config.DefaultRetryPolicy
	}
	
	// Schedule based on type
	switch job.Schedule.Type {
	case ScheduleTypeCron:
		if err := s.scheduleCronJob(job); err != nil {
			return err
		}
	case ScheduleTypeInterval:
		if err := s.scheduleIntervalJob(job); err != nil {
			return err
		}
	case ScheduleTypeEvent:
		if err := s.scheduleEventJob(job); err != nil {
			return err
		}
	case ScheduleTypeOnce:
		if err := s.scheduleOnceJob(job); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported schedule type: %s", job.Schedule.Type)
	}
	
	// Store job
	s.jobs[job.ID] = job
	job.Status = JobStatusScheduled
	
	return nil
}

// UnscheduleJob removes a scheduled job.
func (s *Scheduler) UnscheduleJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job %s not found", jobID)
	}
	
	// Mark job as unscheduled
	job.Status = JobStatusDisabled
	
	// Remove from jobs
	delete(s.jobs, jobID)
	
	return nil
}

// ListJobs returns all scheduled jobs.
func (s *Scheduler) ListJobs() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	jobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	
	return jobs
}

// GetJob returns a specific job.
func (s *Scheduler) GetJob(jobID string) (*Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	job, exists := s.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job %s not found", jobID)
	}
	
	return job, nil
}

// PublishEvent publishes an event that may trigger jobs.
func (s *Scheduler) PublishEvent(event Event) {
	select {
	case s.eventChan <- event:
	default:
		// Event channel full, drop event
		fmt.Printf("Warning: event channel full, dropping event %s\n", event.Type)
	}
}

// RegisterEventHandler registers a handler for a specific event type.
func (s *Scheduler) RegisterEventHandler(eventType string, handler EventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.eventHandlers[eventType] = append(s.eventHandlers[eventType], handler)
}

// scheduleCronJob schedules a cron-based job.
func (s *Scheduler) scheduleCronJob(job *Job) error {
	// Simple cron-like scheduling using goroutines
	// Parse interval from expression (simplified)
	interval := s.parseCronInterval(job.Schedule.Expression)
	if interval == 0 {
		return fmt.Errorf("invalid cron expression: %s", job.Schedule.Expression)
	}
	
	// Create ticker-based job
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				s.executeJob(job)
			}
		}
	}()
	
	job.NextRun = time.Now().Add(interval)
	
	return nil
}

// scheduleIntervalJob schedules an interval-based job.
func (s *Scheduler) scheduleIntervalJob(job *Job) error {
	// Parse interval duration
	interval, err := time.ParseDuration(job.Schedule.Expression)
	if err != nil {
		return fmt.Errorf("invalid interval: %w", err)
	}
	
	// Create ticker-based job
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				s.executeJob(job)
			}
		}
	}()
	
	job.NextRun = time.Now().Add(interval)
	
	return nil
}

// scheduleEventJob schedules an event-triggered job.
func (s *Scheduler) scheduleEventJob(job *Job) error {
	eventType := job.Schedule.Expression
	
	// Register event handler
	s.RegisterEventHandler(eventType, func(event Event) error {
		// Check if job should run based on event
		if s.shouldRunOnEvent(job, event) {
			s.executeJob(job)
		}
		return nil
	})
	
	return nil
}

// scheduleOnceJob schedules a one-time job.
func (s *Scheduler) scheduleOnceJob(job *Job) error {
	// Parse execution time
	execTime, err := time.Parse(time.RFC3339, job.Schedule.Expression)
	if err != nil {
		return fmt.Errorf("invalid time format: %w", err)
	}
	
	// Calculate delay
	delay := time.Until(execTime)
	if delay < 0 {
		return fmt.Errorf("execution time is in the past")
	}
	
	// Schedule one-time execution
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		
		select {
		case <-s.ctx.Done():
			return
		case <-time.After(delay):
			s.executeJob(job)
		}
	}()
	
	job.NextRun = execTime
	
	return nil
}

// executeJob executes a job with retry logic.
func (s *Scheduler) executeJob(job *Job) {
	s.mu.Lock()
	job.Status = JobStatusRunning
	job.LastRun = time.Now()
	job.RunCount++
	s.mu.Unlock()
	
	// Create execution context with timeout
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Minute)
	defer cancel()
	
	// Execute with retry logic
	var err error
	for attempt := 0; attempt <= job.RetryPolicy.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate backoff delay
			delay := time.Duration(float64(job.RetryPolicy.RetryDelay) * 
				pow(job.RetryPolicy.BackoffFactor, float64(attempt-1)))
			time.Sleep(delay)
		}
		
		// Execute handler
		err = job.Handler(ctx, job)
		if err == nil {
			break
		}
		
		fmt.Printf("Job %s attempt %d failed: %v\n", job.ID, attempt+1, err)
	}
	
	// Update job status
	s.mu.Lock()
	if err != nil {
		job.Status = JobStatusFailed
		job.LastError = err
		job.ErrorCount++
	} else {
		job.Status = JobStatusCompleted
		job.ErrorCount = 0
		job.LastError = nil
	}
	
	// Update next run for recurring jobs
	if job.Schedule.Type == ScheduleTypeCron {
		interval := s.parseCronInterval(job.Schedule.Expression)
		if interval > 0 {
			job.NextRun = time.Now().Add(interval)
		}
	}
	s.mu.Unlock()
}

// processEvents processes incoming events.
func (s *Scheduler) processEvents() {
	defer s.wg.Done()
	
	for {
		select {
		case <-s.ctx.Done():
			return
		case event := <-s.eventChan:
			s.handleEvent(event)
		}
	}
}

// handleEvent handles a single event.
func (s *Scheduler) handleEvent(event Event) {
	s.mu.RLock()
	handlers := s.eventHandlers[event.Type]
	s.mu.RUnlock()
	
	for _, handler := range handlers {
		if err := handler(event); err != nil {
			fmt.Printf("Event handler error for %s: %v\n", event.Type, err)
		}
	}
}

// shouldRunOnEvent checks if a job should run based on an event.
func (s *Scheduler) shouldRunOnEvent(job *Job, event Event) bool {
	// Check event metadata against job schedule metadata
	if job.Schedule.Metadata != nil {
		// Add custom filtering logic here
		// For now, always run
		return true
	}
	
	return true
}

// monitorJobs monitors job health and cleanup.
func (s *Scheduler) monitorJobs() {
	defer s.wg.Done()
	
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.cleanupCompletedJobs()
		}
	}
}

// cleanupCompletedJobs removes old completed jobs.
func (s *Scheduler) cleanupCompletedJobs() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	cutoff := time.Now().Add(-24 * time.Hour)
	
	for id, job := range s.jobs {
		if job.Status == JobStatusCompleted && job.LastRun.Before(cutoff) {
			delete(s.jobs, id)
		}
	}
}

// pow calculates x^y for float64.
func pow(x, y float64) float64 {
	result := 1.0
	for i := 0; i < int(y); i++ {
		result *= x
	}
	return result
}

// parseCronInterval parses a simple cron-like expression to a duration.
// This is a simplified version that supports basic intervals.
func (s *Scheduler) parseCronInterval(expr string) time.Duration {
	// Simple parsing for common patterns
	switch expr {
	case "@hourly":
		return 1 * time.Hour
	case "@daily":
		return 24 * time.Hour
	case "@weekly":
		return 7 * 24 * time.Hour
	case "@monthly":
		return 30 * 24 * time.Hour
	case "* * * * *": // Every minute
		return 1 * time.Minute
	case "*/5 * * * *": // Every 5 minutes
		return 5 * time.Minute
	case "*/15 * * * *": // Every 15 minutes
		return 15 * time.Minute
	case "*/30 * * * *": // Every 30 minutes
		return 30 * time.Minute
	case "0 * * * *": // Every hour
		return 1 * time.Hour
	default:
		// Try to parse as duration
		if d, err := time.ParseDuration(expr); err == nil {
			return d
		}
		return 0
	}
}