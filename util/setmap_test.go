package util

import "testing"

func TestSetMap(t *testing.T) {
	sm := NewSetMap[string, int]()
	sm.Add("1", 1)
	sm.Add("1", 2)
	sm.Add("1", 3)
	sm.Add("2", 1)
	sm.Add("2", 2)
	sm.Add("2", 3)
	t.Log(sm.Exists("2", 1))
	t.Log(sm.Exists("1", 3))
	t.Log(sm.Get("1"))
	t.Log(sm.Get("2"))
	sm.Remove("1", 3)
	sm.Remove("2", 1)
	t.Log(sm.Exists("2", 1))
	t.Log(sm.Exists("1", 3))
	t.Log(sm.Exists("2", 2))
	sm.Remove("2", 1)
	t.Log(sm.Exists("2", 2))
	sm.Remove("2", 2)
	t.Log(sm.Exists("2", 2))
	sm.Remove("2", 3)
	t.Log(sm.Get("1"))
	t.Log(sm.Get("2"))
	sm.Add("2", 1)
	t.Log(sm.Exists("2", 2))
	sm.Add("2", 2)
	t.Log(sm.Exists("2", 2))
	sm.Add("2", 3)
}
