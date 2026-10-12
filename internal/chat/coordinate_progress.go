package chat

import (
	"sync"
	"time"
)

// CoordinateProgressSnapshot collects worker events while preserving the
// deterministic order in which the coordinator planned its stages.
type CoordinateProgressSnapshot struct {
	mu     sync.RWMutex
	order  []CoordinateStageID
	stages map[CoordinateStageID]CoordinateStage
}

func NewCoordinateProgressSnapshot(stages []CoordinateStage) *CoordinateProgressSnapshot {
	snapshot := &CoordinateProgressSnapshot{
		order:  makeCoordinateStageOrder(stages),
		stages: make(map[CoordinateStageID]CoordinateStage, len(stages)),
	}
	for _, stage := range stages {
		stage.Dependencies = append([]CoordinateStageID(nil), stage.Dependencies...)
		snapshot.stages[stage.ID] = stage
	}
	return snapshot
}

func makeCoordinateStageOrder(stages []CoordinateStage) []CoordinateStageID {
	order := make([]CoordinateStageID, 0, len(stages))
	for _, stage := range stages {
		order = append(order, stage.ID)
	}
	return order
}

func (s *CoordinateProgressSnapshot) Update(event CoordinateStageEvent) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	stage, ok := s.stages[event.ID]
	if !ok {
		return
	}
	stage.Label = event.Label
	stage.Status = event.Status
	stage.Attempt = event.Attempt
	if event.Status == CoordinateStageRunning || event.Status == CoordinateStageRetrying {
		stage.StartedAt = event.At
		stage.FinishedAt = time.Time{}
	}
	if event.Status == CoordinateStageCompleted || event.Status == CoordinateStageFailed || event.Status == CoordinateStageCancelled {
		stage.FinishedAt = event.At
	}
	s.stages[event.ID] = stage
}

func (s *CoordinateProgressSnapshot) Snapshot() []CoordinateStage {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	stages := make([]CoordinateStage, 0, len(s.order))
	for _, id := range s.order {
		stage, ok := s.stages[id]
		if !ok {
			continue
		}
		stage.Dependencies = append([]CoordinateStageID(nil), stage.Dependencies...)
		stages = append(stages, stage)
	}
	return stages
}
