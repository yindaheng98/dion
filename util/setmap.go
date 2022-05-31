package util

// SetMap is map<K, set<V>>
type SetMap[K, V comparable] struct {
	m map[K]map[V]struct{}
}

func NewSetMap[K, V comparable]() SetMap[K, V] {
	return SetMap[K, V]{make(map[K]map[V]struct{})}
}

// Add a value to the set whose ID is key
func (m SetMap[K, V]) Add(key K, value V) {
	if set, ok := m.m[key]; ok {
		set[value] = struct{}{}
	} else {
		m.m[key] = map[V]struct{}{value: {}}
	}
}

// Remove a value from the set whose ID is key
func (m SetMap[K, V]) Remove(key K, value V) {
	if set, ok := m.m[key]; ok {
		delete(set, value)
		if len(set) <= 0 {
			delete(m.m, key)
		}
	}
}

// Exists show if a value is in the set whose ID is key
func (m SetMap[K, V]) Exists(key K, value V) bool {
	var set map[V]struct{}
	var ok bool
	if set, ok = m.m[key]; ok {
		_, ok = set[value]
	}
	return ok
}

// GetSet get all the values from the set whose ID is key
func (m SetMap[K, V]) GetSet(key K) []V {
	var r []V
	if set, ok := m.m[key]; ok {
		r = make([]V, len(set))
		i := 0
		for value := range set {
			r[i] = value
			i++
		}
	}
	return r
}

// SetMapaMteS consists of a SetMap map<K, set<V>> and a reverse SetMap map<V, set<K>>
// so it can show K和V之间的多对多关系
type SetMapaMteS[K, V comparable] struct {
	SetMap[K, V]
	reverse SetMap[V, K]
}

func NewSetMapaMteS[K, V comparable]() SetMapaMteS[K, V] {
	return SetMapaMteS[K, V]{
		SetMap:  NewSetMap[K, V](),
		reverse: NewSetMap[V, K](),
	}
}

func (m SetMapaMteS[K, V]) Add(key K, value V) {
	m.SetMap.Add(key, value)
	m.reverse.Add(value, key)
}

func (m SetMapaMteS[K, V]) Remove(key K, value V) {
	m.SetMap.Remove(key, value)
	m.reverse.Remove(value, key)
}

func (m SetMapaMteS[K, V]) GetKeys(value V) []K {
	return m.reverse.GetSet(value)
}

// GetUniqueValues only get the key's values that is not exists in other key's set
func (m SetMapaMteS[K, V]) GetUniqueValues(key K) []V {
	var r []V
	if set, ok := m.m[key]; ok {
		for value := range set {
			if len(m.reverse.GetSet(value)) <= 1 { // only have this value?
				r = append(r, value) // that's what I find
			}
		}
	}
	return r
}

// GetUniqueKeys only get the value's keys that is not other value's key
func (m SetMapaMteS[K, V]) GetUniqueKeys(value V) []K {
	var r []K
	if set, ok := m.reverse.m[value]; ok {
		for key := range set {
			if len(m.GetSet(key)) <= 1 { // only have this key?
				r = append(r, key) // that's what I find
			}
		}
	}
	return r
}
