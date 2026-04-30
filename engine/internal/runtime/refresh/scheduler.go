package refresh

import (
	"log"
	"sync"
	"time"
)

type Job struct {
	ModelName string
	Interval  time.Duration
	SyncFn    func() error
	ticker    *time.Ticker
	stopCh    chan struct{}
}

type Scheduler struct {
	mu   sync.Mutex
	jobs map[string]*Job
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		jobs: make(map[string]*Job),
	}
}

func (s *Scheduler) Register(modelName string, interval time.Duration, syncFn func() error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.jobs[modelName]; ok {
		existing.Stop()
	}

	job := &Job{
		ModelName: modelName,
		Interval:  interval,
		SyncFn:    syncFn,
		stopCh:    make(chan struct{}),
	}
	s.jobs[modelName] = job

	job.ticker = time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-job.ticker.C:
				if err := job.SyncFn(); err != nil {
					log.Printf("[REFRESH] %s: sync failed: %v", job.ModelName, err)
				} else {
					log.Printf("[REFRESH] %s: sync completed", job.ModelName)
				}
			case <-job.stopCh:
				return
			}
		}
	}()

	log.Printf("[REFRESH] %s: scheduled every %s", modelName, interval)
}

func (s *Scheduler) RefreshNow(modelName string) error {
	s.mu.Lock()
	job, ok := s.jobs[modelName]
	s.mu.Unlock()

	if !ok {
		return nil
	}
	return job.SyncFn()
}

func (s *Scheduler) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, job := range s.jobs {
		job.Stop()
	}
}

func (s *Scheduler) HasJob(modelName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.jobs[modelName]
	return ok
}

func (j *Job) Stop() {
	if j.ticker != nil {
		j.ticker.Stop()
	}
	select {
	case <-j.stopCh:
	default:
		close(j.stopCh)
	}
}

func ParseRefreshInterval(refresh string) (time.Duration, bool) {
	if refresh == "" || refresh == "startup" || refresh == "never" {
		return 0, false
	}
	d, err := time.ParseDuration(refresh)
	if err != nil {
		log.Printf("[REFRESH] invalid refresh interval %q: %v", refresh, err)
		return 0, false
	}
	if d < time.Minute {
		log.Printf("[REFRESH] refresh interval %q too short, minimum is 1m", refresh)
		return time.Minute, true
	}
	return d, true
}
