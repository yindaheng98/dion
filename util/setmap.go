package util

type SetMap[K, V comparable] struct {
	m map[K]map[V]struct{}
}

func NewSetMap[K, V comparable]() SetMap[K, V] {
	return SetMap[K, V]{make(map[K]map[V]struct{})}
}

func (m SetMap[K, V]) Add(id K, value V) {
	if set, ok := m.m[id]; ok {
		set[value] = struct{}{}
	} else {
		m.m[id] = map[V]struct{}{value: {}}
	}
}

func (m SetMap[K, V]) Remove(id K, value V) {
	if set, ok := m.m[id]; ok {
		delete(set, value)
		if len(set) <= 0 {
			delete(m.m, id)
		}
	}
}

func (m SetMap[K, V]) Exists(id K, value V) bool {
	var set map[V]struct{}
	var ok bool
	if set, ok = m.m[id]; ok {
		_, ok = set[value]
	}
	return ok
}

func (m SetMap[K, V]) Get(id K) (V, bool) {
	if set, ok := m.m[id]; ok {
		for value := range set {
			return value, true
		}
	}
	var value V
	return value, false
}
