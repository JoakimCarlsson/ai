package team

import (
	"context"
	"sync"
	"testing"

	"github.com/joakimcarlsson/ai/agent/team"
)

func TestTeam_CompleteMemberNotFound(_ *testing.T) {
	tm := team.New(team.Config{Name: "test-team"})
	tm.CompleteMember("nonexistent", team.MemberCompleted, "x", "")
}

func TestTeam_GetMemberNotFound(t *testing.T) {
	tm := team.New(team.Config{Name: "test-team"})
	m, ok := tm.GetMember("nonexistent")
	if ok {
		t.Error("expected ok to be false")
	}
	if m != nil {
		t.Error("expected nil member")
	}
}

func TestTeam_FinishMemberNotFound(_ *testing.T) {
	tm := team.New(team.Config{Name: "test-team"})
	tm.FinishMember("nonexistent")
}

func TestTeam_CancelAll(t *testing.T) {
	tm := team.New(team.Config{Name: "test-team"})

	ctx1, c1 := context.WithCancel(context.Background())
	_, _ = tm.AddMember("alice", "t1", c1)

	ctx2, c2 := context.WithCancel(context.Background())
	_, _ = tm.AddMember("bob", "t2", c2)

	tm.CompleteMember(
		"bob",
		team.MemberCompleted,
		"done",
		"",
	)

	tm.CancelAll()

	if ctx1.Err() == nil {
		t.Error("expected alice context to be cancelled")
	}
	if ctx2.Err() != nil {
		t.Error(
			"expected bob context to NOT be cancelled " +
				"(completed members are skipped by CancelAll)",
		)
	}
}

func TestTeam_WaitAll(_ *testing.T) {
	tm := team.New(team.Config{Name: "test-team"})

	_, c1 := context.WithCancel(context.Background())
	_, _ = tm.AddMember("alice", "t1", c1)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		tm.WaitAll()
	}()

	tm.FinishMember("alice")
	wg.Wait()
}

func TestTeam_MaxSizeZeroUnlimited(t *testing.T) {
	tm := team.New(team.Config{Name: "test-team", MaxSize: 0})

	for i := range 10 {
		_, c := context.WithCancel(context.Background())
		name := "member-" + string(rune('a'+i))
		_, err := tm.AddMember(name, "task", c)
		if err != nil {
			t.Fatalf("unexpected error adding member %d: %v", i, err)
		}
	}

	if tm.ActiveCount() != 10 {
		t.Errorf(
			"expected 10 active, got %d",
			tm.ActiveCount(),
		)
	}
}

func TestTeam_NameAndMaxSize(t *testing.T) {
	tm := team.New(team.Config{Name: "my-team", MaxSize: 5})
	if tm.Name() != "my-team" {
		t.Errorf("expected name %q, got %q", "my-team", tm.Name())
	}
	if tm.MaxSize() != 5 {
		t.Errorf("expected max size 5, got %d", tm.MaxSize())
	}
}

func TestTeam_CompleteMemberWithError(t *testing.T) {
	tm := team.New(team.Config{Name: "test-team"})

	_, cancel := context.WithCancel(context.Background())
	_, _ = tm.AddMember("alice", "task", cancel)

	tm.CompleteMember(
		"alice",
		team.MemberFailed,
		"",
		"something broke",
	)

	m, ok := tm.GetMember("alice")
	if !ok {
		t.Fatal("member not found")
	}
	if m.Status != team.MemberFailed {
		t.Errorf(
			"expected status %q, got %q",
			team.MemberFailed,
			m.Status,
		)
	}
	if m.Error != "something broke" {
		t.Errorf(
			"expected error %q, got %q",
			"something broke",
			m.Error,
		)
	}
}

func TestChannelMailbox_ReadAfterClose(t *testing.T) {
	mb := team.NewChannelMailbox()
	mb.RegisterRecipient("alice")

	_ = mb.Send(
		context.Background(),
		team.Message{
			From:    "bob",
			To:      "alice",
			Content: "hello",
		},
	)

	_ = mb.Close()

	_, err := mb.Read(context.Background(), "alice")
	if err == nil {
		t.Error("expected error reading from closed mailbox")
	}
}

func TestChannelMailbox_RegisterIdempotent(t *testing.T) {
	mb := team.NewChannelMailbox()
	defer mb.Close()

	mb.RegisterRecipient("alice")
	mb.RegisterRecipient("alice")

	_ = mb.Send(
		context.Background(),
		team.Message{
			From:    "bob",
			To:      "alice",
			Content: "single",
		},
	)

	msgs, err := mb.Read(context.Background(), "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 1 {
		t.Errorf("expected 1 message, got %d", len(msgs))
	}
}
