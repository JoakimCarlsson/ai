package team

import (
	"testing"

	"github.com/joakimcarlsson/ai/agent/team"
)

func TestTaskBoard_CompleteNotFound(t *testing.T) {
	tb := team.NewTaskBoard()
	err := tb.Complete("nonexistent", "alice", "result")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestTaskBoard_ListEmpty(t *testing.T) {
	tb := team.NewTaskBoard()
	tasks := tb.List()
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestTaskBoard_MultipleCreate(t *testing.T) {
	tb := team.NewTaskBoard()

	t1 := tb.Create("task 1", "alice")
	t2 := tb.Create("task 2", "bob")
	t3 := tb.Create("task 3", "carol")

	if t1.ID == t2.ID || t2.ID == t3.ID {
		t.Error("expected unique task IDs")
	}

	tasks := tb.List()
	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tasks))
	}
}
