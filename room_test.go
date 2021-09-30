package wsroom_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/Lambels/wsroom"
)

func BenchmarkCreateRoom(b *testing.B) {
	store := wsroom.NewRuntimeStore()
	for i := 0; i < b.N; i++ {
		r, _ := store.New(strconv.Itoa(i), 1024, time.Second*5)
		fmt.Println(r)
	}
}

func BenchmarkGetRoom(b *testing.B) {
	b.StopTimer()
	store := wsroom.NewRuntimeStore()
	for i := 0; i < b.N; i++ {
		r, _ := store.New(strconv.Itoa(i), 1024, time.Second*5)
		fmt.Println(r)
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		r, _ := store.Get(strconv.Itoa(i))
		fmt.Println(r)
	}
	b.StopTimer()
}
