package repository

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"homework/internal/model"
)

// ErrIDExhausted means the uint32 identifier space is exhausted.
var ErrIDExhausted = errors.New("resource identifier space exhausted")

// Store is an in-memory repository safe for concurrent requests.
type Store struct {
	mu      sync.RWMutex
	variant model.Variant
	items   map[uint32]model.Resource
	nextID  uint32
}

// New creates an empty store for one router instance.
func New(variant model.Variant) *Store {
	return &Store{
		variant: variant,
		items:   make(map[uint32]model.Resource),
		nextID:  1,
	}
}

// Create stores a resource and returns a copy of the created item.
func (s *Store) Create(fields map[string]any) (model.Resource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.nextID == 0 {
		return model.Resource{}, ErrIDExhausted
	}
	id := s.nextID
	s.nextID++
	now := time.Now().UTC()
	item := model.Resource{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		Fields:    clone(fields),
	}
	s.items[id] = item
	return item.Clone(), nil
}

// Get returns a copy of one resource.
func (s *Store) Get(id uint32) (model.Resource, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[id]
	if !ok {
		return model.Resource{}, false
	}
	return item.Clone(), true
}

// Update replaces the complete resource and changes only updated_at.
func (s *Store) Update(id uint32, fields map[string]any) (model.Resource, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.items[id]
	if !ok {
		return model.Resource{}, false
	}
	current.Fields = clone(fields)
	current.UpdatedAt = time.Now().UTC()
	s.items[id] = current
	return current.Clone(), true
}

// Delete removes one resource.
func (s *Store) Delete(id uint32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return false
	}
	delete(s.items, id)
	return true
}

// List returns resources sorted by ID and optionally filtered by the variant filter field.
func (s *Store) List(filter string) []model.Resource {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]uint32, 0, len(s.items))
	for id := range s.items {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	result := make([]model.Resource, 0, len(ids))
	for _, id := range ids {
		item := s.items[id]
		if filter != "" {
			value, _ := item.Fields[s.variant.FilterField()].(string)
			if !strings.EqualFold(value, filter) {
				continue
			}
		}
		result = append(result, item.Clone())
	}
	return result
}

func clone(item map[string]any) map[string]any {
	result := make(map[string]any, len(item))
	for key, value := range item {
		result[key] = value
	}
	return result
}
