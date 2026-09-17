package cloud

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"netcompanion/internal/history"
)

type flakyPusher struct {
	calls     atomic.Int32
	failFirst int32
}

func (f *flakyPusher) Push(context.Context, history.Snapshot) error {
	n := f.calls.Add(1)
	if n <= f.failFirst {
		return errors.New("échec temporaire")
	}
	return nil
}

func TestPushBestEffortRetriesThenSucceeds(t *testing.T) {
	p := &flakyPusher{failFirst: 2} // échoue 2 fois, réussit à la 3e
	PushBestEffort(p, history.Snapshot{ID: "x"}, nil)
	if got := p.calls.Load(); got != 3 {
		t.Fatalf("attendu 3 tentatives, obtenu %d", got)
	}
}

func TestPushBestEffortGivesUpAfterThree(t *testing.T) {
	p := &flakyPusher{failFirst: 99}                  // échoue toujours
	PushBestEffort(p, history.Snapshot{ID: "x"}, nil) // ne doit pas paniquer/bloquer
	if got := p.calls.Load(); got != 3 {
		t.Fatalf("attendu 3 tentatives max, obtenu %d", got)
	}
}
