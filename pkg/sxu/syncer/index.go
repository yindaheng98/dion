package syncer

import (
	"sort"
	"strings"
)

type IndexData interface {
	// Key return the key of this item
	Key() string
	// Compare compare if two IndexData is the same
	Compare(IndexData) bool
}

type indexTuple struct {
	data    IndexData
	checked bool
}

type Index struct {
	index map[string]*indexTuple
}

func (index *Index) reset() {
	for _, tuple := range index.index {
		tuple.checked = false
	}
}

// Add add a data
func (index *Index) Add(data IndexData) {
	index.index[data.Key()] = &indexTuple{
		data:    data,
		checked: false,
	}
}

// Del delete a data
func (index *Index) Del(data IndexData) {
	delete(index.index, data.Key())
}

// Exist check if a data exists
func (index *Index) Exist(data IndexData) bool {
	if tuple, ok := index.index[data.Key()]; ok {
		return tuple.data.Compare(data)
	}
	return false
}

// Update update the index
// and shows the difference between index and the input list
func (index *Index) Update(list []IndexData) (add []IndexData, del []IndexData) {
	index.reset()

	// Find those data in input list but not in index, add it or replace it
	for _, data := range list {
		if tuple, ok := index.index[data.Key()]; ok { // Check if the key exists
			// Exits?
			if tuple.data.Compare(data) { // Compare the value
				// Also same?
				tuple.checked = true // do nothing
			} else { // Value not same?
				del = append(del, tuple.data) // delete the exists
				add = append(add, data)       // add the new
				tuple.data = data             // replace it
				tuple.checked = true
			}
		} else { //key not exists?
			add = append(add, data) // just add the new
			index.index[data.Key()] = &indexTuple{
				data:    data,
				checked: true,
			}
		}
	}

	// Find those data in index but not in input list, delete it
	for key, tuple := range index.index {
		if !tuple.checked { // not exists in input list?
			del = append(del, tuple.data) // delete it
			delete(index.index, key)
		}
	}

	return
}

type indexDataList []IndexData

func (s indexDataList) Len() int {
	return len(s)
}

func (s indexDataList) Less(i, j int) bool {
	return strings.Compare(s[i].Key(), s[j].Key()) < 0
}

func (s indexDataList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Gather gather all the data and output it in key order
func (index *Index) Gather() []IndexData {
	var list = make(indexDataList, len(index.index))
	i := 0
	for _, d := range index.index {
		list[i] = d.data
		i++
	}
	sort.Sort(list)
	return list
}

// TODO: Cache the Gather(), if the index not same, just return the cache
