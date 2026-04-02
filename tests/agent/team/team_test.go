package team

import (
	"context"
	"testing"

	"github.com/joakimcarlsson/ai/agent/team"
)

func TestTeam_AddMember(t *testing.T) {
	tm := team.New(team.Config{Name: "test-team", MaxSize: 3})

	_, cancel := context.WithCancel(context.Background())
	m, err := tm.AddMember("alice", "research", cancel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name != "alice" {
		t.Errorf("expected name %q, got %q", "alice", m.Name)
	}
	if m.Status != team.MemberActive {
		t.Errorf("expected status %q, got %q", team.MemberActive, m.Status)
	}
	if tm.ActiveCount() != 1 {
		t.Errorf("expected active count 1, got %d", tm.ActiveCount())
	}
}

func TestTeam_AddDuplicateMember(t *testing.T) {
	tm := team.New(team.Config{Name: "test-team"})

	_, cancel := context.WithCancel(context.Background())
	_, _ = tm.AddMember("alice", "task1", cancel)

	_, cancel2 := context.WithCancel(context.Background())
	_, err := tm.AddMember("alice", "task2", cancel2)
	if err == nil {
		t.Error("expected error for duplicate member name")
	}
	cancel2()
}

func TestTeam_MaxSize(t *testing.T) {
	tm := team.New(team.Config{Name: "test-team", MaxSize: 2})

	_, c1 := context.WithCancel(context.Background())
	_, _ = tm.AddMember("alice", "t1", c1)

	_, c2 := context.WithCancel(context.Background())
	_, _ = tm.AddMember("bob", "t2", c2)

	_, c3 := context.WithCancel(context.Background())
	_, err := tm.AddMember("carol", "t3", c3)
	if err == nil {
		t.Error("expected error when exceeding max size")
	}
	c3()
}

func TestTeam_CompleteMember(t *testing.T) {
	tm := team.New(team.Config{Name: "test-team"})

	_, cancel := context.WithCancel(context.Background())
	_, _ = tm.AddMember("alice", "research", cancel)

	tm.CompleteMember("alice", team.MemberCompleted, "done", "")

	m, ok := tm.GetMember("alice")
	if !ok {
		t.Fatal("member not found")
	}
	if m.Status != team.MemberCompleted {
		t.Errorf("expected status %q, got %q", team.MemberCompleted, m.Status)
	}
	if m.Result != "done" {
		t.Errorf("expected result %q, got %q", "done", m.Result)
	}
}

func TestTeam_StopMember(t *testing.T) {
	tm := team.New(team.Config{Name: "test-team"})

	ctx, cancel := context.WithCancel(context.Background())
	_, _ = tm.AddMember("alice", "task", cancel)

	err := tm.StopMember("alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.Err() == nil {
		t.Error("expected context to be cancelled")
	}
}

func TestTeam_StopMemberNotFound(t *testing.T) {
	tm := team.New(team.Config{Name: "test-team"})
	err := tm.StopMember("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent member")
	}
}

func TestTeam_ListMembers(t *testing.T) {
	tm := team.New(team.Config{Name: "test-team"})

	_, c1 := context.WithCancel(context.Background())
	_, _ = tm.AddMember("alice", "t1", c1)

	_, c2 := context.WithCancel(context.Background())
	_, _ = tm.AddMember("bob", "t2", c2)

	members := tm.ListMembers()
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
}

func TestTeam_ContextPropagation(t *testing.T) {
	tm := team.New(team.Config{Name: "ctx-team"})
	ctx := context.Background()

	if team.FromContext(ctx) != nil {
		t.Error("expected nil team from empty context")
	}

	ctx = team.WithContext(ctx, tm)
	if team.FromContext(ctx) != tm {
		t.Error("expected team from context")
	}

	if team.IsLead(ctx) {
		t.Error("expected non-lead context")
	}

	ctx = team.WithLeadContext(ctx)
	if !team.IsLead(ctx) {
		t.Error("expected lead context")
	}
}
