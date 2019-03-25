package spidy

import (
	"sync"
)

type Storage struct {
	visitedURLs map[uint64]bool
	lock        *sync.RWMutex
}

func (s *Storage) Init() error {
	if s.visitedURLs == nil {
		s.visitedURLs = make(map[uint64]bool)
	}
	if s.lock == nil {
		s.lock = &sync.RWMutex{}
	}
	return nil
}

func (s *Storage) Visited(requestID uint64) error {
	s.lock.Lock()
	s.visitedURLs[requestID] = true
	s.lock.Unlock()
	return nil
}

func (s *Storage) IsVisited(requestID uint64) (bool, error) {
	s.lock.RLock()
	visited := s.visitedURLs[requestID]
	s.lock.RUnlock()
	return visited, nil
}

