package datamodel_test

import (
	"marathon-sim/datamodel"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFastQueueEnqueue(t *testing.T) {
	queue := datamodel.NewFastQueue()

	queue.Enqueue("item1")
	queue.Enqueue("item2")
	queue.Enqueue("item3")

	assert.True(t, queue.Contains("item1"))
	assert.True(t, queue.Contains("item2"))
	assert.True(t, queue.Contains("item3"))
}

func TestFastQueueDequeue(t *testing.T) {
	queue := datamodel.NewFastQueue()

	queue.Enqueue("item1")
	queue.Enqueue("item2")
	queue.Enqueue("item3")

	item, ok := queue.Dequeue()
	assert.True(t, ok)
	assert.Equal(t, "item1", item)

	item, ok = queue.Dequeue()
	assert.True(t, ok)
	assert.Equal(t, "item2", item)

	item, ok = queue.Dequeue()
	assert.True(t, ok)
	assert.Equal(t, "item3", item)

	// Queue is empty
	item, ok = queue.Dequeue()
	assert.False(t, ok)
	assert.Equal(t, "", item)
}

func TestFastQueueContains(t *testing.T) {
	queue := datamodel.NewFastQueue()

	queue.Enqueue("item1")
	queue.Enqueue("item2")
	queue.Enqueue("item3")

	assert.True(t, queue.Contains("item1"))
	assert.True(t, queue.Contains("item2"))
	assert.True(t, queue.Contains("item3"))

	assert.False(t, queue.Contains("item4"))
	assert.False(t, queue.Contains("item5"))
}
