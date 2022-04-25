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

type DisorderSetItemReplaceTuple struct {
	Old DisorderSetItem
	New DisorderSetItem
}

func (s *DisorderSet) reset() {
	for _, tuple := range s.index {
		tuple.checked = false
	}
}

// Update update the index
// and shows the difference between index and the input list
func (s *DisorderSet) Update(list []DisorderSetItem) (add []DisorderSetItem, del []DisorderSetItem, replace []DisorderSetItemReplaceTuple) {
	s.reset()

	// Find those data in input list but not in index, add it or replace it
	for _, data := range list {
		if tuple, ok := s.index[data.Key()]; ok { // Check if the key exists
			// Exits?
			if tuple.data.Compare(data) { // Compare the value
				// Also same?
				tuple.checked = true // do nothing
			} else { // Value not same?
				// should replace
				replace = append(replace, DisorderSetItemReplaceTuple{Old: tuple.data, New: data})
				tuple.data = data.Clone() // replace it
				tuple.checked = true
			}
		} else { //key not exists?
			add = append(add, data) // just add the new
			s.index[data.Key()] = &itemTuple{
				data:    data.Clone(),
				checked: true,
			}
		}
	}

	// Find those data in index but not in input list, delete it
	for key, tuple := range s.index {
		if !tuple.checked { // not exists in input list?
			del = append(del, tuple.data) // delete it
			delete(s.index, key)
		}
	}

	return
}

// IsSame just the compare the index and the input and return if is the same
func (s *DisorderSet) IsSame(list []DisorderSetItem) bool {
	s.reset()

	// Find those data in input list but not in index, add it or replace it
	for _, data := range list {
		if tuple, ok := s.index[data.Key()]; ok { // Check if the key exists
			// Exits?
			if tuple.data.Compare(data) { // Compare the value
				// Also same?
				tuple.checked = true // do nothing
			} else { // Value not same?
				return false
			}
		} else { //key not exists?
			return false
		}
	}

	// Find those data in index but not in input list, delete it
	for _, tuple := range s.index {
		if !tuple.checked { // not exists in input list?
			return false
		}
	}
	return true
}

type DisorderSetItemList []DisorderSetItem

func NewDisorderSetFromList(list DisorderSetItemList) *DisorderSet {
	set := &DisorderSet{index: make(map[string]*itemTuple, len(list))}
	for _, item := range list {
		set.Add(item)
	}
	return set
}
