package util

type StringDisorderSetItem string

func (s StringDisorderSetItem) Key() string {
	return string(s)
}

func (s StringDisorderSetItem) Compare(item DisorderSetItem) bool {
	return item == s
}

func (s StringDisorderSetItem) Clone() DisorderSetItem {
	return s
}

type Strings []string

func (ss Strings) ToDisorderSetItemList() DisorderSetItemList[StringDisorderSetItem] {
	list := make([]StringDisorderSetItem, len(ss))
	for i, s := range ss {
		list[i] = StringDisorderSetItem(s)
	}
	return list
}
