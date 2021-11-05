package dolo

import (
	"github.com/boggydigital/nod"
)

var maxConcurrentOps = 4

type concurrencyCounter struct {
	currentItem         int
	totalItems          int
	remainingItems      int
	remainingConcurrent int
	tpw                 nod.TotalProgressWriter
}

func newConcurrencyCounter(total int, tpw nod.TotalProgressWriter) *concurrencyCounter {
	if total > 1 &&
		tpw != nil {
		tpw.TotalInt(total)
	}
	return &concurrencyCounter{
		currentItem:         0,
		totalItems:          total,
		remainingItems:      total,
		remainingConcurrent: maxConcurrentOps,
		tpw:                 tpw,
	}
}

func (ct *concurrencyCounter) complete() {
	ct.remainingItems--
	ct.remainingConcurrent++
	if ct.totalItems > 1 &&
		ct.tpw != nil {
		ct.tpw.Increment()
	}
}

func (ct *concurrencyCounter) scheduleNext() {
	ct.currentItem++
}

func (ct *concurrencyCounter) canSchedule() bool {
	return ct.currentItem < ct.totalItems
}

func (ct *concurrencyCounter) hasRemaining() bool {
	return ct.remainingItems > 0
}

func (ct *concurrencyCounter) current() int {
	return ct.currentItem
}

func (ct *concurrencyCounter) allowedConcurrent() int {
	return ct.remainingConcurrent
}

func (ct *concurrencyCounter) flushRemainingConcurrent() {
	ct.remainingConcurrent = 0
}
