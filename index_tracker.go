package dolo

import (
	"sync"
)

var maxAvailability = 4
var mutex = sync.Mutex{}

type indexTracker struct {
	requested  map[int]bool
	processing map[int]bool
	completed  map[int]bool
	available  int
}

func newIndexTracker(total int) *indexTracker {

	mutex.Lock()
	requestedItems := make(map[int]bool, total)
	for i := 0; i < total; i++ {
		requestedItems[i] = true
	}
	mutex.Unlock()

	return &indexTracker{
		requested:  requestedItems,
		processing: make(map[int]bool, total),
		completed:  make(map[int]bool, total),
		available:  maxAvailability,
	}
}

func (it *indexTracker) complete(index int) {

	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := it.processing[index]; ok {
		delete(it.processing, index)
		it.completed[index] = true
	}

	it.available++
	if it.available > maxAvailability {
		it.available = maxAvailability
	}
}

func (it *indexTracker) nextRequested() int {
	mutex.Lock()
	defer mutex.Unlock()

	for index, ok := range it.requested {
		if !ok {
			continue
		}
		return index
	}

	return -1
}

func (it *indexTracker) processNext() int {
	nr := it.nextRequested()
	mutex.Lock()
	defer mutex.Unlock()

	if nr > -1 {
		delete(it.requested, nr)
		it.processing[nr] = true
	}
	it.available--

	return nr
}

func (it *indexTracker) canProcess() bool {
	mutex.Lock()
	defer mutex.Unlock()
	return it.available > 0
}

func (it *indexTracker) hasRequested() bool {
	mutex.Lock()
	defer mutex.Unlock()
	return len(it.requested) > 0
}

func (it *indexTracker) hasProcessing() bool {
	mutex.Lock()
	defer mutex.Unlock()
	return len(it.processing) > 0
}

func (it *indexTracker) hasWork() bool {
	return it.hasRequested() || it.hasProcessing()
}
