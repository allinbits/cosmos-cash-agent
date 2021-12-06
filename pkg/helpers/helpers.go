package helpers

import (
	"time"
)

func DelayExec(delay int, fn func()) {
	t1 := time.NewTicker(time.Duration(delay) * time.Second)
	go func() {
		<-t1.C
		fn()
	}()
}
