package util

import (
	"testing"
	"time"
)

func TestSingleExec(t *testing.T) {
	e := NewSingleExec()
	for i := 0; i < 100; i++ {
		go func(i int) {
			e.Do(func() {
				for j := 0; j < 5; j++ {
					t.Logf("%d is running %d\n", i, j)
					<-time.After(1 * time.Second)
				}
			})
		}(i)
		<-time.After(100 * time.Millisecond)
	}
	<-time.After(10 * time.Second)
}
