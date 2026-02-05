package queue_test

import (
	"testing"

	"github.com/Radon10043/cloud/src/generator/queue"
)

func TestPop(t *testing.T) {
	jstr := `
[
	{"type": "syscall", "name": "ioctl"},
	{"type": "init_syscall", "name": "openat"},
	{"type": "struct", "name": "test"}
]
	`
	pq, err := queue.NewTaskQueueFromJson(jstr)
	if err != nil {
		t.Fatalf("Failed to create TaskQueue: %v", err)
	}
	elem, err := pq.Pop()
	if err != nil {
		t.Fatalf("Failed to pop from TaskQueue: %v", err)
	}
	if elem.Name != "openat" {
		t.Errorf("Expected init_syscall 'openat', got '%s'", elem.Name)
	}
	elem, err = pq.Pop()
	if err != nil {
		t.Fatalf("Failed to pop from TaskQueue: %v", err)
	}
	if elem.Name != "test" {
		t.Errorf("Expected struct 'test', got '%s'", elem.Name)
	}
}
