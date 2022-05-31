package util

import (
	"context"
	"math/rand"
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
					<-time.After(time.Duration(rand.Int31n(1000)) * time.Millisecond)
				}
			})
		}(i)
		<-time.After(time.Duration(rand.Int31n(100)) * time.Millisecond)
	}
	<-time.After(10 * time.Second)
}

func TestSingleLatestExec(t *testing.T) {
	e := SingleLatestExec{}
	for i := 0; i < 100; i++ {
		go func(i int) {
			e.Do(func(ctx context.Context) {
				for j := 0; j < 5; j++ {
					select {
					case <-ctx.Done():
						t.Logf("%d exit\n", i)
						return
					default:
					}
					t.Logf("%d is running %d\n", i, j)
					select {
					case <-ctx.Done():
						t.Logf("%d exit\n", i)
						return
					case <-time.After(time.Duration(rand.Int31n(500)) * time.Millisecond):
					}
				}
			})
		}(i)
		<-time.After(time.Duration(rand.Int31n(1000)) * time.Millisecond)
	}
}

func TestSingleWaitExec(t *testing.T) {
	e := NewSingleWaitExec(context.Background())
	for i := 0; i < 100; i++ {
		go func(i int) {
			e.Do(func() {
				for j := 0; j < 5; j++ {
					t.Logf("%d is running %d\n", i, j)
					<-time.After(time.Duration(rand.Int31n(1000)) * time.Millisecond)
				}
			})
			t.Logf("%d is ok\n", i)
		}(i)
		<-time.After(time.Duration(rand.Int31n(100)) * time.Millisecond)
	}
	<-time.After(10 * time.Second)
}
