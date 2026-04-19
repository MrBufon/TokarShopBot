package collections

import "sync"

type MutexMap[K comparable, V any] struct {
	data map[K]V
	mu   sync.RWMutex
}

func NewMutexMap[K comparable, V any](size ...int) *MutexMap[K, V] {
	capacity := 0

	if len(size) > 0 && size[0] > 0 {
		capacity = size[0]
	}

	return &MutexMap[K, V]{
		data: make(map[K]V, capacity),
	}
}

func (m *MutexMap[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, ok := m.data[key]
	return value, ok
}

func (m *MutexMap[K, V]) Set(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = value
}

func (m *MutexMap[K, V]) Delete(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
}
