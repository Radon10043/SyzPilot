package queue

import (
	"container/heap"
	"encoding/json"
	"sync"
)

type TaskHeapElemType int

// Define task types with priorities
const (
	TaskHeapElemTypeUnknown TaskHeapElemType = iota
	TaskHeapElemTypeInitSyscall
	TaskHeapElemTypeStruct
	TaskHeapElemTypeUnion
	TaskHeapElemTypeSyscall
)

// Priority returns the priority of the TaskHeapElemType, lower value means higher priority
func (t TaskHeapElemType) Priority() int {
	return int(t)
}

// String returns the string representation of the TaskHeapElemType
func (t TaskHeapElemType) String() string {
	switch t {
	case TaskHeapElemTypeInitSyscall:
		return "init_syscall"
	case TaskHeapElemTypeStruct:
		return "struct"
	case TaskHeapElemTypeUnion:
		return "union"
	case TaskHeapElemTypeSyscall:
		return "syscall"
	default:
		return "unknown"
	}
}

// ParseTaskHeapElemType parses a string and returns the corresponding TaskHeapElemType
func ParseTaskHeapElemType(s string) TaskHeapElemType {
	switch s {
	case "init_syscall":
		return TaskHeapElemTypeInitSyscall
	case "struct":
		return TaskHeapElemTypeStruct
	case "union":
		return TaskHeapElemTypeUnion
	case "syscall":
		return TaskHeapElemTypeSyscall
	default:
		return TaskHeapElemTypeUnknown
	}
}

// TaskHeapElem represents a task with its name and type
type TaskHeapElem struct {
	Name string           `json:"name"`
	Type TaskHeapElemType `json:"type"`
}

// taskHeap implements a priority queue for TaskHeapElem
type taskHeap []*TaskHeapElem

// Len returns the length of the taskHeap
func (h taskHeap) Len() int {
	return len(h)
}

// Less compares two elements in the taskHeap based on their priority.
// If priorities are equal, it compares their names lexicographically.
func (h taskHeap) Less(i, j int) bool {
	p1 := h[i].Type.Priority()
	p2 := h[j].Type.Priority()
	if p1 != p2 {
		return p1 < p2
	}
	return h[i].Name < h[j].Name
}

// Swap swaps two elements in the taskHeap
func (h taskHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

// Push adds an element to the taskHeap
func (h *taskHeap) Push(x any) {
	*h = append(*h, x.(*TaskHeapElem))
}

// Pop removes and returns the last element from the taskHeap
func (h *taskHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// Peek returns the first element of the taskHeap without removing it
func (h *taskHeap) Peek() *TaskHeapElem {
	return (*h)[0]
}

// Empty checks if the taskHeap is empty
func (h *taskHeap) Empty() bool {
	return len(*h) == 0
}

// Clear removes all elements from the taskHeap
func (h *taskHeap) Clear() {
	*h = (*h)[:0]
}

// TaskQueue is a thread-safe wrapper around taskHeap
type TaskQueue struct {
	impl *taskHeap
	mu   sync.Mutex
}

// TaskQueueElem represents a task with its name and type
type TaskQueueElem struct {
	Name string
	Type string
}

// Push adds an element to the TaskQueue
func (q *TaskQueue) Push(elem *TaskQueueElem) {
	q.mu.Lock()
	defer q.mu.Unlock()
	heap.Push(q.impl, &TaskHeapElem{
		Name: elem.Name,
		Type: ParseTaskHeapElemType(elem.Type),
	})
}

// Pop removes and returns the highest priority element from the TaskQueue
func (q *TaskQueue) Pop() *TaskQueueElem {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.impl.Empty() {
		return nil
	}
	elem := heap.Pop(q.impl).(*TaskHeapElem)
	return &TaskQueueElem{
		Name: elem.Name,
		Type: elem.Type.String(),
	}
}

// Peek returns the highest priority element from the TaskQueue without removing it
func (q *TaskQueue) Peek() *TaskQueueElem {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.impl.Empty() {
		return nil
	}
	tmp := q.impl.Peek()
	return &TaskQueueElem{
		Name: tmp.Name,
		Type: tmp.Type.String(),
	}
}

// Empty checks if the TaskQueue is empty
func (q *TaskQueue) Empty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.impl.Empty()
}

// Clear removes all elements from the TaskQueue
func (q *TaskQueue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.impl.Clear()
}

// Len returns the number of elements in the TaskQueue
func (q *TaskQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.impl.Len()
}

// Slice returns a slice of TaskQueueElem representing the elements in the TaskQueue
func (q *TaskQueue) Slice() []TaskQueueElem {
	q.mu.Lock()
	defer q.mu.Unlock()
	elements := make([]TaskQueueElem, q.impl.Len())
	for i, elem := range *q.impl {
		elements[i].Name = elem.Name
		elements[i].Type = elem.Type.String()
	}
	return elements
}

// Json returns the JSON representation of the TaskQueue
func (q *TaskQueue) Json() (string, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	elements := make([]TaskQueueElem, q.impl.Len())
	for i, elem := range *q.impl {
		elements[i].Name = elem.Name
		elements[i].Type = elem.Type.String()
	}
	jbytes, err := json.MarshalIndent(elements, "", "\t")
	if err != nil {
		return "", err
	}
	return string(jbytes), nil
}

// NewTaskQueue creates an empty TaskQueue
func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		impl: &taskHeap{},
	}
}

// NewTaskQueueFromJson creates a TaskQueue from a JSON string
func NewTaskQueueFromJson(jstr string) (*TaskQueue, error) {
	var elements []TaskQueueElem
	err := json.Unmarshal([]byte(jstr), &elements)
	if err != nil {
		return nil, err
	}

	q := NewTaskQueue()
	for _, elem := range elements {
		q.Push(&TaskQueueElem{
			Name: elem.Name,
			Type: elem.Type,
		})
	}
	return q, nil
}
