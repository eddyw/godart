//go:build dart

package sync

// A Map is a map that many goroutines read and write.
//
// All Go code runs in 1 Dart isolate, and a goroutine gives up control only at
// a block operation. Thus this Map is a plain map with the API of Go, and it
// needs no lock.
//
// The zero value is an empty Map, and it is ready to use.
type Map struct {
	m map[any]any
}

// Load gives the value of a key. The flag is false when the Map holds no such
// key.
func (m *Map) Load(key any) (any, bool) {
	v, ok := m.m[key]
	return v, ok
}

// Store writes the value of a key.
func (m *Map) Store(key, value any) {
	if m.m == nil {
		m.m = map[any]any{}
	}
	m.m[key] = value
}

// Clear drops each key.
func (m *Map) Clear() { m.m = nil }

// LoadOrStore gives the value of a key. It writes the new value when the Map
// holds no such key, and the flag reports a key that the Map already held.
func (m *Map) LoadOrStore(key, value any) (any, bool) {
	if v, ok := m.m[key]; ok {
		return v, true
	}
	m.Store(key, value)
	return value, false
}

// LoadAndDelete gives the value of a key and drops the key.
func (m *Map) LoadAndDelete(key any) (any, bool) {
	v, ok := m.m[key]
	if ok {
		delete(m.m, key)
	}
	return v, ok
}

// Delete drops a key.
func (m *Map) Delete(key any) { delete(m.m, key) }

// Swap writes the value of a key and gives the value that the key held.
func (m *Map) Swap(key, value any) (any, bool) {
	v, ok := m.m[key]
	m.Store(key, value)
	return v, ok
}

// CompareAndSwap writes the new value of a key when the key holds the old
// value.
func (m *Map) CompareAndSwap(key, old, new any) bool {
	v, ok := m.m[key]
	if !ok || v != old {
		return false
	}
	m.m[key] = new
	return true
}

// CompareAndDelete drops a key when the key holds the old value.
func (m *Map) CompareAndDelete(key, old any) bool {
	v, ok := m.m[key]
	if !ok || v != old {
		return false
	}
	delete(m.m, key)
	return true
}

// Range calls f for each key of the Map. It stops when f gives false.
//
// The loop reads a copy of the keys, thus a Store of f reaches no later pass.
func (m *Map) Range(f func(key, value any) bool) {
	keys := make([]any, 0, len(m.m))
	for k := range m.m {
		keys = append(keys, k)
	}
	for _, k := range keys {
		v, ok := m.m[k]
		if !ok {
			continue
		}
		if !f(k, v) {
			return
		}
	}
}
