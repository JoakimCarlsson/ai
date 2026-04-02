package team

import (
	"testing"

	"github.com/joakimcarlsson/ai/agent/team"
)

func TestTaskBoard_CreateAndList(t *testing.T) {
	tb := team.NewTaskBoard()

	task := tb.Create("research topic", "alice")
	if task.ID == "" {
		t.Fatal("expected non-empty task ID")
	}
	if task.Title != "research topic" {
		t.Errorf("expected title %q, got %q", "research topic", task.Title)
	}
	if task.Status != team.TaskOpen {
		t.Errorf("expected status %q, got %q", team.TaskOpen, task.Status)
	}
	if task.CreatedBy != "alice" {
		t.Errorf("expected created_by %q, got %q", "alice", task.CreatedBy)
	}

	tasks := tb.List()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
}

func TestTaskBoard_ClaimAndComplete(t *testing.T) {
	tb := team.NewTaskBoard()
	task := tb.Create("write report", "lead")

	if err := tb.Claim(task.ID, "bob"); err != nil {
		t.Fatalf("unexpected claim error: %v", err)
	}

	tasks := tb.List()
	var found *team.Task
	for _, tsk := range tasks {
		if tsk.ID == task.ID {
			found = tsk
			break
		}
	}
	if found == nil {
		t.Fatal("task not found after claim")
	}
	if found.Status != team.TaskClaimed {
		t.Errorf("expected status %q, got %q", team.TaskClaimed, found.Status)
	}
	if found.Assignee != "bob" {
		t.Errorf("expected assignee %q, got %q", "bob", found.Assignee)
	}

	if err := tb.Complete(task.ID, "bob", "done"); err != nil {
		t.Fatalf("unexpected complete error: %v", err)
	}

	tasks = tb.List()
	for _, tsk := range tasks {
		if tsk.ID == task.ID {
			found = tsk
			break
		}
	}
	if found.Status != team.TaskCompleted {
		t.Errorf("expected status %q, got %q", team.TaskCompleted, found.Status)
	}
	if found.Result != "done" {
		t.Errorf("expected result %q, got %q", "done", found.Result)
	}
}

func TestTaskBoard_ClaimAlreadyClaimed(t *testing.T) {
	tb := team.NewTaskBoard()
	task := tb.Create("task", "lead")
	_ = tb.Claim(task.ID, "alice")

	err := tb.Claim(task.ID, "bob")
	if err == nil {
		t.Error("expected error claiming already-claimed task")
	}
}

func TestTaskBoard_CompleteWrongAssignee(t *testing.T) {
	tb := team.NewTaskBoard()
	task := tb.Create("task", "lead")
	_ = tb.Claim(task.ID, "alice")

	err := tb.Complete(task.ID, "bob", "result")
	if err == nil {
		t.Error("expected error completing task with wrong assignee")
	}
}

func TestTaskBoard_ClaimNotFound(t *testing.T) {
	tb := team.NewTaskBoard()
	err := tb.Claim("nonexistent", "alice")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}
