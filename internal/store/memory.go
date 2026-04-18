package store

import (
	"errors"
	"sort"
	"sync"

	"github.com/Modificator/readlater-wip/internal/model"
)

var ErrNotFound = errors.New("not found")

type MemoryRepository struct {
	mu      sync.RWMutex
	tasks   map[string]*model.ArchiveTask
	records map[string]*model.ArchiveRecord
	rules   map[string]*model.ExtractionRule
	targets map[string]*model.GiteaTarget
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		tasks:   map[string]*model.ArchiveTask{},
		records: map[string]*model.ArchiveRecord{},
		rules:   map[string]*model.ExtractionRule{},
		targets: map[string]*model.GiteaTarget{},
	}
}

func (m *MemoryRepository) SaveTask(task *model.ArchiveTask) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cloned := *task
	m.tasks[task.ID] = &cloned
}

func (m *MemoryRepository) GetTask(id string) (*model.ArchiveTask, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tasks[id]
	if !ok {
		return nil, ErrNotFound
	}
	cloned := *t
	return &cloned, nil
}

func (m *MemoryRepository) FindTaskByURL(url string) (*model.ArchiveTask, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.tasks {
		if t.URL == url {
			cloned := *t
			return &cloned, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryRepository) SaveRecord(record *model.ArchiveRecord) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cloned := *record
	m.records[record.ID] = &cloned
}

func (m *MemoryRepository) SaveRule(rule *model.ExtractionRule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cloned := *rule
	m.rules[rule.ID] = &cloned
}

func (m *MemoryRepository) GetRule(id string) (*model.ExtractionRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rules[id]
	if !ok {
		return nil, ErrNotFound
	}
	cloned := *r
	return &cloned, nil
}

func (m *MemoryRepository) ListRules() []*model.ExtractionRule {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*model.ExtractionRule, 0, len(m.rules))
	for _, r := range m.rules {
		cloned := *r
		out = append(out, &cloned)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Priority == out[j].Priority {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].Priority > out[j].Priority
	})
	return out
}

func (m *MemoryRepository) SaveGiteaTarget(target *model.GiteaTarget) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cloned := *target
	m.targets[target.ID] = &cloned
}

func (m *MemoryRepository) ListGiteaTargets() []*model.GiteaTarget {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*model.GiteaTarget, 0, len(m.targets))
	for _, t := range m.targets {
		cloned := *t
		out = append(out, &cloned)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (m *MemoryRepository) GetGiteaTargetByAlias(alias string) (*model.GiteaTarget, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.targets {
		if t.Alias == alias {
			cloned := *t
			return &cloned, nil
		}
	}
	return nil, ErrNotFound
}
