package team

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Config configures a team that a lead agent manages.
type Config struct {
	Name    string
	MaxSize int
}

// MemberStatus represents the lifecycle state of a teammate.
type MemberStatus string

// MemberActive, MemberCompleted, MemberFailed, MemberStopped are the possible member statuses.
const (
	MemberActive    MemberStatus = "active"
	MemberCompleted MemberStatus = "completed"
	MemberFailed    MemberStatus = "failed"
	MemberStopped   MemberStatus = "stopped"
)

// Member holds metadata about a running teammate.
type Member struct {
	ID       string
	Name     string
	Status   MemberStatus
	Task     string
	JoinedAt time.Time
	EndedAt  time.Time
	Result   string
	Error    string
	Done     chan struct{}
	cancel   context.CancelFunc
}

// Team manages a roster of concurrently-running peer agents.
type Team struct {
	mu        sync.RWMutex
	config    Config
	roster    map[string]*Member
	Mailbox   Mailbox
	TaskBoard *TaskBoard
	wg        sync.WaitGroup
	idGen     atomic.Int64
}

// New creates a new Team with a default channel mailbox and task board.
func New(config Config) *Team {
	return &Team{
		config:    config,
		roster:    make(map[string]*Member),
		Mailbox:   NewChannelMailbox(),
		TaskBoard: NewTaskBoard(),
	}
}

// Name returns the team's name.
func (t *Team) Name() string {
	return t.config.Name
}

// MaxSize returns the team's maximum member count.
func (t *Team) MaxSize() int {
	return t.config.MaxSize
}

// AddMember registers a new teammate in the roster and mailbox.
func (t *Team) AddMember(
	name, task string,
	cancel context.CancelFunc,
) (*Member, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.roster[name]; exists {
		return nil, fmt.Errorf("teammate %q already exists", name)
	}

	active := 0
	for _, m := range t.roster {
		if m.Status == MemberActive {
			active++
		}
	}
	if t.config.MaxSize > 0 && active >= t.config.MaxSize {
		return nil, fmt.Errorf("team is at max capacity (%d)", t.config.MaxSize)
	}

	id := fmt.Sprintf("teammate-%d", t.idGen.Add(1))
	m := &Member{
		ID:       id,
		Name:     name,
		Status:   MemberActive,
		Task:     task,
		JoinedAt: time.Now(),
		Done:     make(chan struct{}),
		cancel:   cancel,
	}
	t.roster[name] = m
	t.Mailbox.RegisterRecipient(name)
	t.wg.Add(1)

	return m, nil
}

// CompleteMember updates a member's status, result, and end time.
func (t *Team) CompleteMember(
	name string,
	status MemberStatus,
	result, errMsg string,
) {
	t.mu.Lock()
	defer t.mu.Unlock()

	m, ok := t.roster[name]
	if !ok {
		return
	}
	m.Status = status
	m.Result = result
	m.Error = errMsg
	m.EndedAt = time.Now()
}

// FinishMember closes the member's Done channel and decrements the WaitGroup.
func (t *Team) FinishMember(name string) {
	t.mu.RLock()
	m, ok := t.roster[name]
	t.mu.RUnlock()
	if ok {
		close(m.Done)
		t.wg.Done()
	}
}

// GetMember returns a snapshot of the named member.
func (t *Team) GetMember(name string) (*Member, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	m, ok := t.roster[name]
	if !ok {
		return nil, false
	}
	snapshot := *m
	return &snapshot, true
}

// ListMembers returns a snapshot of all teammates.
func (t *Team) ListMembers() []*Member {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]*Member, 0, len(t.roster))
	for _, m := range t.roster {
		snapshot := *m
		result = append(result, &snapshot)
	}
	return result
}

// ActiveCount returns the number of currently active teammates.
func (t *Team) ActiveCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	count := 0
	for _, m := range t.roster {
		if m.Status == MemberActive {
			count++
		}
	}
	return count
}

// StopMember cancels a running teammate by name.
func (t *Team) StopMember(name string) error {
	t.mu.RLock()
	m, ok := t.roster[name]
	t.mu.RUnlock()

	if !ok {
		return fmt.Errorf("teammate %q not found", name)
	}
	m.cancel()
	return nil
}

// CancelAll cancels every active teammate's context.
func (t *Team) CancelAll() {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, m := range t.roster {
		if m.Status == MemberActive {
			m.cancel()
		}
	}
}

// WaitAll blocks until every teammate has finished.
func (t *Team) WaitAll() {
	t.wg.Wait()
}

type teamKey struct{}
type teamLeadKey struct{}

// WithContext stores the team in the context.
func WithContext(ctx context.Context, t *Team) context.Context {
	return context.WithValue(ctx, teamKey{}, t)
}

// FromContext retrieves the team from the context.
func FromContext(ctx context.Context) *Team {
	t, _ := ctx.Value(teamKey{}).(*Team)
	return t
}

// WithLeadContext marks the context as belonging to the team lead.
func WithLeadContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, teamLeadKey{}, true)
}

// IsLead returns true if the context belongs to the team lead.
func IsLead(ctx context.Context) bool {
	v, _ := ctx.Value(teamLeadKey{}).(bool)
	return v
}
