package util

import (
	"sort"
	"strings"
)

type DisorderSetItem interface {
	// Key return the key of this item
	Key() string
	// Compare compare if two IndexData is the same
	Compare(DisorderSetItem) bool
	// Clone clone return a copy of this struct
	Clone() DisorderSetItem
}

type itemTuple[ItemType DisorderSetItem] struct {
	data    ItemType
	checked bool
}

// DisorderSet is a disorder set, thread-UNSAFE!
type DisorderSet[ItemType DisorderSetItem] struct {
	index map[string]*itemTuple[ItemType]
}

func NewDisorderSet[ItemType DisorderSetItem]() *DisorderSet[ItemType] {
	return &DisorderSet[ItemType]{index: make(map[string]*itemTuple[ItemType])}
}

// Add add a data
func (s *DisorderSet[ItemType]) Add(data ItemType) {
	s.index[data.Key()] = &itemTuple[ItemType]{
		data:    data.Clone().(ItemType),
		checked: false,
	}
}

// Del delete a data
func (s *DisorderSet[ItemType]) Del(data ItemType) {
	delete(s.index, data.Key())
}

// Exist check if a data exists
func (s *DisorderSet[ItemType]) Exist(data ItemType) bool {
	if tuple, ok := s.index[data.Key()]; ok {
		return tuple.data.Compare(data)
	}
	return false
}

type DisorderSetItemReplaceTuple[ItemType DisorderSetItem] struct {
	Old ItemType
	New ItemType
}

func (s *DisorderSet[ItemType]) reset() {
	for _, tuple := range s.index {
		tuple.checked = false
	}
}

// Update update the index
// and shows the difference between index and the input list
func (s *DisorderSet[ItemType]) Update(list []ItemType) (add []ItemType, del []ItemType, replace []DisorderSetItemReplaceTuple[ItemType]) {
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
				replace = append(replace, DisorderSetItemReplaceTuple[ItemType]{Old: tuple.data, New: data})
				tuple.data = data.Clone().(ItemType) // replace it
				tuple.checked = true
			}
		} else { //key not exists?
			add = append(add, data) // just add the new
			s.index[data.Key()] = &itemTuple[ItemType]{
				data:    data.Clone().(ItemType),
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
func (s *DisorderSet[ItemType]) IsSame(list []ItemType) bool {
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

type DisorderSetItemList[ItemType DisorderSetItem] []ItemType

func NewDisorderSetFromList[ItemType DisorderSetItem](list DisorderSetItemList[ItemType]) *DisorderSet[ItemType] {
	set := &DisorderSet[ItemType]{index: make(map[string]*itemTuple[ItemType], len(list))}
	for _, item := range list {
		set.Add(item)
	}
	return set
}

func (s DisorderSetItemList[ItemType]) Len() int {
	return len(s)
}

func (s DisorderSetItemList[ItemType]) Less(i, j int) bool {
	return strings.Compare(s[i].Key(), s[j].Key()) < 0
}

func (s DisorderSetItemList[ItemType]) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Sort gather all the data and output it in key order
func (s *DisorderSet[ItemType]) Sort() DisorderSetItemList[ItemType] {
	var list = make(DisorderSetItemList[ItemType], len(s.index))
	i := 0
	for _, d := range s.index {
		list[i] = d.data.Clone().(ItemType)
		i++
	}
	sort.Sort(list)
	return list
}

// TODO: Cache the Gather(), if the index not same, just return the cache
