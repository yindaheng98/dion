package util

type DisorderSetItem interface {
	// Key return the key of this item
	Key() string
	// Compare compare if two IndexData is the same
	Compare(DisorderSetItem) bool
	// Clone clone return a copy of this struct
	Clone() DisorderSetItem
}

type itemTuple struct {
	data    DisorderSetItem
	checked bool
}

// DisorderSet is a disorder set, thread-UNSAFE!
type DisorderSet struct {
	index map[string]*itemTuple
}

func NewDisorderSet() *DisorderSet {
	return &DisorderSet{index: make(map[string]*itemTuple)}
}

// Add add a data
func (s *DisorderSet) Add(data DisorderSetItem) {
	s.index[data.Key()] = &itemTuple{
		data:    data.Clone(),
		checked: false,
	}
}

// Del delete a data
func (s *DisorderSet) Del(data DisorderSetItem) {
	delete(s.index, data.Key())
}

// Exist check if a data exists
func (s *DisorderSet) Exist(data DisorderSetItem) bool {
	if tuple, ok := s.index[data.Key()]; ok {
		return tuple.data.Compare(data)
	}
	return false
}
