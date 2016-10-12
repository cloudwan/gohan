package nmutex

import (
	"sync"
)

// NamedMutex provides a locking mechanism with names
// Use New() function to get an instance of NamedMutex
type NamedMutex struct {
	outer sync.Mutex
	sets  map[string]*set
}

// New creates a new instance of NamedMutex
func New() *NamedMutex {
	return &NamedMutex{
		sets: make(map[string]*set),
	}
}

// Lock gets the lock for a name.
// When there is another goroutine that already has the lock
// for the name, this method blocks.
// Otherwithe, you immediately get the lock for the name.
// The return value is a function to release the lock, which
// must be called to allow other routines to secure the lock.
// When the lock for a name has been released and
// there is no other gouroutines waiting for the lock,
// the internal resource for the name is automatically released.
func (m *NamedMutex) Lock(name string) UnlockFunc {
	m.outer.Lock()
	s, ok := m.sets[name]
	if !ok {
		s = new(set)
		m.sets[name] = s
	}
	s.count++
	m.outer.Unlock()

	s.lock.Lock()

	return func() {
		m.outer.Lock()
		defer m.outer.Unlock()

		s.count--
		if s.count == 0 {
			delete(m.sets, name)
		}
		s.lock.Unlock()
	}
}

// UnlockFunc is returned by Lock() to release the lock
type UnlockFunc func()

type set struct {
	count int
	lock  sync.Mutex
}
