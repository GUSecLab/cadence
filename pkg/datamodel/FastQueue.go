package datamodel

import "sync"

type FastQueue struct {
	mutex sync.Mutex
	items map[string]struct{}
}

func NewFastQueue() *FastQueue {
	return &FastQueue{
		items: make(map[string]struct{}),
	}
}

func (q *FastQueue) Enqueue(item string) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.items[item] = struct{}{}
}

func (q *FastQueue) Dequeue() (string, bool) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	for item := range q.items {
		delete(q.items, item)
		return item, true
	}

	return "", false // Queue is empty
}

func (q *FastQueue) Contains(item string) bool {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	_, exists := q.items[item]
	return exists
}
