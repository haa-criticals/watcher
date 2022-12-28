package watcher

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type mockWatcherClient struct {
	fRequestRegister func(ctx context.Context, address, key string) (*RegisterResponse, error)
	fAckNode         func(ctx context.Context, address, key string, node *Peer) (*Peer, error)
	fRequestVote     func(ctx context.Context, address, candidate string, term int64) (*Vote, error)
}

func (m *mockWatcherClient) RequestRegister(ctx context.Context, address, key string) (*RegisterResponse, error) {
	return m.fRequestRegister(ctx, address, key)
}

func (m *mockWatcherClient) AckNode(ctx context.Context, address, key string, node *Peer) (*Peer, error) {
	return m.fAckNode(ctx, address, key, node)
}

func (m *mockWatcherClient) RequestVote(ctx context.Context, address, candidate string, term int64) (*Vote, error) {
	return m.fRequestVote(ctx, address, candidate, term)
}

func TestWatcher(t *testing.T) {
	t.Run("Should notify leader down after 3 seconds counting from last received beat", func(t *testing.T) {
		leaderDownTriggered := false
		w := &Watcher{
			config: Config{
				HeartBeatCheckInterval: 5 * time.Millisecond,
				MaxDelayForElection:    1,
			},
			lastReceivedBeat: time.Now(),
			OnLeaderDown: func(nodes []*Peer, lastReceivedBeat time.Time) {
				leaderDownTriggered = true
			},
		}

		go w.StartHeartBeatChecking()
		time.Sleep(15 * time.Millisecond)
		assert.True(t, leaderDownTriggered)
	})

	t.Run("Should not notify leader down if received beat in time", func(t *testing.T) {
		leaderDownTriggered := false
		w := &Watcher{
			config: Config{
				HeartBeatCheckInterval: 5 * time.Millisecond,
				MaxDelayForElection:    5,
			},
			lastReceivedBeat: time.Now(),
			OnLeaderDown: func(nodes []*Peer, lastReceivedBeat time.Time) {
				leaderDownTriggered = true
			},
		}

		go w.StartHeartBeatChecking()
		time.Sleep(5 * time.Millisecond)
		w.OnReceiveHeartBeat("", 0, time.Now())
		time.Sleep(5 * time.Millisecond)
		w.OnReceiveHeartBeat("", 0, time.Now())
		assert.False(t, leaderDownTriggered)
	})

	t.Run("Should not start checking heart beat if already started", func(t *testing.T) {
		leaderDownTriggeredCount := 0
		w := &Watcher{
			config: Config{
				HeartBeatCheckInterval: 1 * time.Millisecond,
				MaxDelayForElection:    1,
			},
			lastReceivedBeat: time.Now(),
			OnLeaderDown: func(nodes []*Peer, lastReceivedBeat time.Time) {
				leaderDownTriggeredCount = leaderDownTriggeredCount + 1
			},
		}

		go w.StartHeartBeatChecking()
		go w.StartHeartBeatChecking()
		time.Sleep(5 * time.Millisecond)
		assert.Equal(t, 1, leaderDownTriggeredCount)
	})

	t.Run("Should not notify leader down if stopped checking heart beat", func(t *testing.T) {
		leaderDownTriggered := false
		w := &Watcher{
			config: Config{
				HeartBeatCheckInterval: 10 * time.Millisecond,
				MaxDelayForElection:    5,
			},
			lastReceivedBeat:      time.Now(),
			doneHeartBeatChecking: make(chan struct{}),
			OnLeaderDown: func(nodes []*Peer, lastReceivedBeat time.Time) {
				leaderDownTriggered = true
			},
		}

		go w.StartHeartBeatChecking()
		time.Sleep(5 * time.Millisecond)
		w.StopHeartBeatChecking()
		time.Sleep(10 * time.Millisecond)
		assert.False(t, leaderDownTriggered)
	})

	t.Run("Should not notify leader down if min notification interval is not passed yet", func(t *testing.T) {
		leaderDownTriggeredCount := 0
		w := &Watcher{
			config: Config{
				HeartBeatCheckInterval: 5 * time.Millisecond,
				MaxDelayForElection:    1,
			},
			lastReceivedBeat: time.Now(),
			OnLeaderDown: func(nodes []*Peer, lastReceivedBeat time.Time) {
				leaderDownTriggeredCount++
			},
		}

		go w.StartHeartBeatChecking()
		time.Sleep(20 * time.Millisecond)
		assert.Equal(t, 1, leaderDownTriggeredCount)
	})

	t.Run("Should create an election if leader is down", func(t *testing.T) {
		w := &Watcher{
			client: &mockWatcherClient{
				fRequestVote: func(ctx context.Context, address, candidate string, term int64) (*Vote, error) {
					return &Vote{
						Granted: true,
					}, nil
				},
			},
			config: Config{
				HeartBeatCheckInterval: 5 * time.Millisecond,
				MaxDelayForElection:    1,
			},
			lastReceivedBeat: time.Now(),
			nodes: []*Peer{
				{"node1"},
				{"node2"},
			},
		}

		go w.StartHeartBeatChecking()
		time.Sleep(20 * time.Millisecond)
		assert.NotNil(t, w.election)
	})

	t.Run("Should start an election if leader is down", func(t *testing.T) {
		w := &Watcher{
			client: &mockWatcherClient{
				fRequestVote: func(ctx context.Context, address, candidate string, term int64) (*Vote, error) {
					return &Vote{
						Granted: true,
					}, nil
				},
			},
			config: Config{
				HeartBeatCheckInterval: 5 * time.Millisecond,
				MaxDelayForElection:    1,
			},
			lastReceivedBeat: time.Now(),
			nodes: []*Peer{
				{"192.168.0.10"},
				{"191.168.0.11"},
			},
		}

		go w.StartHeartBeatChecking()
		time.Sleep(20 * time.Millisecond)
		assert.NotNil(t, w.election)
		assert.False(t, w.election.startedAt.IsZero())
	})

	t.Run("Election should start with random delay", func(t *testing.T) {
		w := &Watcher{
			client: &mockWatcherClient{
				fRequestVote: func(ctx context.Context, address, candidate string, term int64) (*Vote, error) {
					return &Vote{
						Granted: true,
					}, nil
				},
			},
			config: Config{MaxDelayForElection: 30},
			nodes: []*Peer{
				{"192.168.0.10"},
			},
		}

		timeRef := time.Now()
		w.startElection()
		time.Sleep(50 * time.Millisecond)
		assert.True(t, w.election.startedAt.After(timeRef))

		timeDiff := w.election.startedAt.Sub(timeRef)

		timeRef = time.Now()
		w.startElection()
		time.Sleep(50 * time.Millisecond)
		assert.True(t, w.election.startedAt.After(timeRef))

		timeDiff2 := w.election.startedAt.Sub(timeRef)

		assert.NotEqual(t, timeDiff, timeDiff2)
	})

	t.Run("Should increment the term when start an election", func(t *testing.T) {
		w := &Watcher{
			client: &mockWatcherClient{
				fRequestVote: func(ctx context.Context, address, candidate string, term int64) (*Vote, error) {
					return &Vote{
						Granted: true,
					}, nil
				},
			},
			term:   1,
			config: Config{MaxDelayForElection: 1},
			nodes: []*Peer{
				{"192.168.0.10"},
			},
		}

		w.startElection()
		time.Sleep(2 * time.Millisecond)
		assert.Equal(t, int64(2), w.term)
	})

	t.Run("Should vote for itself when start an election", func(t *testing.T) {
		w := &Watcher{
			client: &mockWatcherClient{
				fRequestVote: func(ctx context.Context, address, candidate string, term int64) (*Vote, error) {
					return &Vote{
						Granted: true,
					}, nil
				},
			},
			Address: "192.168.0.1",
			term:    1,
			config:  Config{MaxDelayForElection: 1},
			nodes: []*Peer{
				{"192.168.0.10"},
			},
		}

		w.startElection()
		time.Sleep(2 * time.Millisecond)
		assert.Equal(t, w.Address, w.votedFor)
	})

	t.Run("Should call request vote if leader is down", func(t *testing.T) {

		requestedVote := false
		w := &Watcher{
			client: &mockWatcherClient{
				fRequestVote: func(ctx context.Context, address, candidate string, term int64) (*Vote, error) {
					requestedVote = true
					return &Vote{
						Granted: true,
					}, nil
				},
			},
			config: Config{MaxDelayForElection: 1},
			nodes:  []*Peer{{"node1"}},
		}

		w.startElection()
		time.Sleep(5 * time.Millisecond)
		assert.NotNil(t, w.election)
		assert.True(t, requestedVote)

	})

	t.Run("Should deny vote if term is lower than current term", func(t *testing.T) {
		w := &Watcher{
			term: 2,
		}

		r := w.OnReceiveVoteRequest(&Candidate{Term: 1})
		assert.False(t, r.Granted)
	})

	t.Run("Should deny vote if already voted for another node", func(t *testing.T) {
		w := &Watcher{
			term:     2,
			votedFor: "192.168.0.10",
		}

		r := w.OnReceiveVoteRequest(&Candidate{Term: 2})
		assert.False(t, r.Granted)
	})

	t.Run("Should return the current term when vote is denied", func(t *testing.T) {
		w := &Watcher{
			term:     2,
			votedFor: "other",
		}

		r := w.OnReceiveVoteRequest(&Candidate{Term: 2})
		assert.False(t, r.Granted)
		assert.Equal(t, int64(2), r.Term)
	})

	t.Run("Should grant vote", func(t *testing.T) {
		w := &Watcher{
			term:   2,
			config: Config{},
		}

		r := w.OnReceiveVoteRequest(&Candidate{Term: 2})
		assert.True(t, r.Granted)
	})

	t.Run("Should set votedFor when grant vote", func(t *testing.T) {
		w := &Watcher{
			term:   2,
			config: Config{},
		}

		r := w.OnReceiveVoteRequest(&Candidate{2, "192.168.0.10"})
		assert.True(t, r.Granted)
		assert.Equal(t, "192.168.0.10", w.votedFor)
	})

	t.Run("Update the term when vote is granted", func(t *testing.T) {
		w := &Watcher{
			term:   2,
			config: Config{},
		}

		r := w.OnReceiveVoteRequest(&Candidate{3, "192.168.0.10"})
		assert.True(t, r.Granted)
		assert.Equal(t, int64(3), w.term)
	})
}
