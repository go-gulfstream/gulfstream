package main

import "sync/atomic"

type someLocalIndexInMem struct {
	count uint32
}

func (i *someLocalIndexInMem) Increment() {
	atomic.AddUint32(&i.count, 1)
}

func (i *someLocalIndexInMem) Load() int {
	return int(atomic.LoadUint32(&i.count))
}
