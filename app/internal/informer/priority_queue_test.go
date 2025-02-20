package informer

import (
	"container/heap"
	"testing"
	"time"
)

func TestPriorityQueue_Pop_Push(t *testing.T) {
	pq := make(PriorityQueue, 0)
	heap.Push(&pq, &Item{value: "a", priority: time.Now().Add(time.Hour)})
	heap.Push(&pq, &Item{value: "b", priority: time.Now().Add(time.Hour * 2)})
	heap.Push(&pq, &Item{value: "c", priority: time.Now().Add(time.Minute * 10)})
	heap.Push(&pq, &Item{value: "d", priority: time.Now().Add(time.Minute * 5)})

	if pq[0].value != "d" {
		t.Errorf("expected d, got %s", pq[0].value)
	}
	if pq[0].index != 0 {
		t.Errorf("expected 0, got %d", pq[0].index)
	}
	i := heap.Pop(&pq).(*Item)
	if i.value != "d" {
		t.Errorf("expected a, got %s", i.value)
	}

	if pq[0].value != "c" {
		t.Errorf("expected c, got %s", pq[0].value)
	}
	if pq[0].index != 0 {
		t.Errorf("expected 0, got %d", pq[0].index)
	}
	i = heap.Pop(&pq).(*Item)
	if i.value != "c" {
		t.Errorf("expected c, got %s", i.value)
	}

	if pq[0].value != "a" {
		t.Errorf("expected a, got %s", pq[0].value)
	}
	if pq[0].index != 0 {
		t.Errorf("expected 0, got %d", pq[0].index)
	}
	i = heap.Pop(&pq).(*Item)
	if i.value != "a" {
		t.Errorf("expected b, got %s", i.value)
	}

	heap.Push(&pq, &Item{value: "e", priority: time.Now().Add(time.Minute)})
	if pq[0].value != "e" {
		t.Errorf("expected e, got %s", pq[0].value)
	}
	if pq[0].index != 0 {
		t.Errorf("expected 0, got %d", pq[0].index)
	}

	heap.Push(&pq, &Item{value: "f", priority: time.Now().Add(10 * time.Hour)})
	if pq[0].value != "e" {
		t.Errorf("expected e, got %s", pq[0].value)
	}
}

func TestPriorityQueue_update(t *testing.T) {
	pq := make(PriorityQueue, 0)
	heap.Push(&pq, &Item{value: "a", priority: time.Now().Add(time.Hour)})
	heap.Push(&pq, &Item{value: "b", priority: time.Now().Add(time.Hour * 2)})

	pq.update(pq[0], pq[0].value, time.Now().Add(time.Hour*3))
	if pq[0].value != "b" {
		t.Errorf("expected b, got %s", pq[0].value)
	}
	pq.update(pq[1], pq[1].value, time.Now().Add(time.Hour*1))
	if pq[0].value != "a" {
		t.Errorf("expected a, got %s", pq[0].value)
	}
	if pq[0].index != 0 {
		t.Errorf("expected 0, got %d", pq[0].index)
	}
	if pq[1].value != "b" {
		t.Errorf("expected b, got %s", pq[1].value)
	}
	if pq[1].index != 1 {
		t.Errorf("expected 1, got %d", pq[1].index)
	}
}
