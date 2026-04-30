package refresh

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestParseRefreshInterval_Valid(t *testing.T) {
	tests := []struct {
		input    string
		wantDur  time.Duration
		wantOK   bool
	}{
		{"5m", 5 * time.Minute, true},
		{"1h", time.Hour, true},
		{"30m", 30 * time.Minute, true},
		{"2h30m", 150 * time.Minute, true},
	}

	for _, tt := range tests {
		d, ok := ParseRefreshInterval(tt.input)
		if ok != tt.wantOK {
			t.Errorf("ParseRefreshInterval(%q): ok=%v, want %v", tt.input, ok, tt.wantOK)
		}
		if d != tt.wantDur {
			t.Errorf("ParseRefreshInterval(%q): duration=%v, want %v", tt.input, d, tt.wantDur)
		}
	}
}

func TestParseRefreshInterval_NonSchedulable(t *testing.T) {
	tests := []string{"", "startup", "never"}
	for _, input := range tests {
		_, ok := ParseRefreshInterval(input)
		if ok {
			t.Errorf("ParseRefreshInterval(%q) should return false", input)
		}
	}
}

func TestParseRefreshInterval_MinimumEnforced(t *testing.T) {
	d, ok := ParseRefreshInterval("10s")
	if !ok {
		t.Fatal("expected ok=true for short interval (should be clamped)")
	}
	if d < time.Minute {
		t.Errorf("expected minimum 1m, got %v", d)
	}
}

func TestScheduler_RegisterAndRefreshNow(t *testing.T) {
	s := NewScheduler()
	defer s.StopAll()

	var callCount int32
	s.Register("test_model", 10*time.Minute, func() error {
		atomic.AddInt32(&callCount, 1)
		return nil
	})

	if !s.HasJob("test_model") {
		t.Error("expected job to be registered")
	}

	if err := s.RefreshNow("test_model"); err != nil {
		t.Fatalf("RefreshNow failed: %v", err)
	}

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestScheduler_RefreshNow_NoJob(t *testing.T) {
	s := NewScheduler()
	if err := s.RefreshNow("nonexistent"); err != nil {
		t.Errorf("RefreshNow for nonexistent should return nil, got %v", err)
	}
}

func TestScheduler_StopAll(t *testing.T) {
	s := NewScheduler()

	s.Register("m1", 10*time.Minute, func() error { return nil })
	s.Register("m2", 10*time.Minute, func() error { return nil })

	s.StopAll()
}

func TestScheduler_ReRegister(t *testing.T) {
	s := NewScheduler()
	defer s.StopAll()

	var count1, count2 int32
	s.Register("model", 10*time.Minute, func() error {
		atomic.AddInt32(&count1, 1)
		return nil
	})

	s.Register("model", 10*time.Minute, func() error {
		atomic.AddInt32(&count2, 1)
		return nil
	})

	s.RefreshNow("model")

	if atomic.LoadInt32(&count1) != 0 {
		t.Error("old job should not be called after re-register")
	}
	if atomic.LoadInt32(&count2) != 1 {
		t.Errorf("new job should be called, got %d", count2)
	}
}
