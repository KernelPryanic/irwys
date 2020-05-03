package irwys

import "sync"

// SynMap structure.
// Implements map of interfaces with adaptation to multithreading.
type SynMap struct {
	data map[interface{}]interface{}
	lock *sync.RWMutex
}

// NewSynMap creates an object of SynMap structure.
func NewSynMap() SynMap {
	lock := sync.RWMutex{}
	data := map[interface{}]interface{}{}
	synMap := SynMap{data, &lock}

	return synMap
}

// Get an object from vault.
func (m SynMap) Get(key interface{}) interface{} {
	(*m.lock).RLock()
	defer (*m.lock).RUnlock()
	return m.data[key]
}

// Put an object into vault.
func (m SynMap) Put(key interface{}, value interface{}) {
	(*m.lock).Lock()
	defer (*m.lock).Unlock()
	m.data[key] = value
}

// Delete an object from vault.
func (m SynMap) Delete(key interface{}) {
	(*m.lock).Lock()
	defer (*m.lock).Unlock()
	delete(m.data, key)
}

// Exist checks if an object in vault.
func (m SynMap) Exist(key interface{}) bool {
	(*m.lock).RLock()
	defer (*m.lock).RUnlock()

	_, ok := m.data[key]

	return ok
}

// Iterate brings possibility to iterate over vault
// by returning map of interfaces.
func (m SynMap) Iterate() map[interface{}]interface{} {
	return m.data
}

// Len returns length of vault.
func (m SynMap) Len() int {
	return len(m.data)
}
