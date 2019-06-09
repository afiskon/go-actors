package system

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/afiskon/go-actors/actor"
	"github.com/afiskon/go-actors/actor/errors"
	"testing"
	"time"
)

func TestSystemStartStop(t *testing.T) {
	t.Parallel()
	system := New()
	system.AwaitTermination()
}

func TestSystemSendNonExistingActor(t *testing.T) {
	t.Parallel()
	system := New()
	err := system.Send(actor.Pid(123), "hello")
	require.Equal(t, err, errors.InvalidPid)
	err = system.SendPriority(actor.Pid(123), "hello")
	require.Equal(t, err, errors.InvalidPid)
	system.AwaitTermination()
}

type TestSpawnActor struct {
	ch chan struct{}
}

func (a *TestSpawnActor) Receive(actor.Message) (actor.Actor, error) {
	a.ch <- struct {}{}
	return a, errors.Terminate
}

func TestSystemSpawnSendTerminate(t *testing.T) {
	t.Parallel()
	system := New()
	received := make(chan struct{}, 1)
	pid := system.Spawn(func(system actor.System, pid actor.Pid) (state actor.Actor, limit int) {
		return &TestSpawnActor{ch: received}, 0
	})
	err := system.Send(pid, "hello")
	require.NoError(t, err)
	<-received
	for {
		err = system.Send(pid, "hello")
		if err == errors.InvalidPid {
			break
		}
		// still terminating ..
		time.Sleep(10 * time.Millisecond)
	}
	system.AwaitTermination()
}

type TestPriorityActor struct {
	regular_received chan struct{}
	regular_unblock chan struct{}
	priority_ch chan struct{}
}

type RegularMessage struct {}
type PriorityMessage struct {}
type TerminateMessage struct {}

func (a *TestPriorityActor) Receive(message actor.Message) (actor.Actor, error) {
	switch v := message.(type) {
	case RegularMessage:
		a.regular_received <- struct{}{}
		a.regular_unblock <- struct{}{}
		return a, nil
	case PriorityMessage:
		a.priority_ch <- struct{}{}
		return a, nil
	case TerminateMessage:
		return a, errors.Terminate
	default:
		return a, fmt.Errorf("Don't know what to do with %T", v)
	}
}

func TestSystemSendPriority(t *testing.T) {
	t.Parallel()
	system := New()
	regular_received := make(chan struct{})
	regular_unblock := make(chan struct{})
	priority_ch := make(chan struct{})
	pid := system.Spawn(func(system actor.System, pid actor.Pid) (state actor.Actor, limit int) {
		return &TestPriorityActor{
			regular_received: regular_received,
			regular_unblock: regular_unblock,
			priority_ch: priority_ch,
		}, 0
	})
	err := system.Send(pid, RegularMessage{})
	require.NoError(t, err)

	// make sure the message was received
	<-regular_received

	// send two more messages
	err = system.Send(pid, RegularMessage{})
	require.NoError(t, err)
	err = system.SendPriority(pid, PriorityMessage{})
	require.NoError(t, err)

	// unblock Receive
	<-regular_unblock
	// make sure PriorityMessage arrived before 2nd RegularMessage
	<-priority_ch
	// unblock Receive once again
	<-regular_received
	<-regular_unblock

	err = system.Send(pid, TerminateMessage{})
	require.NoError(t, err)
	system.AwaitTermination()
}

type StateWaitingStash struct {
	ch chan int
}

type StateWaitingUnstash struct {
	ch chan int
}

type StateProcessingStash struct {
	ch chan int
}

type MsgStash struct {}
type MsgUnstash struct {}

func (a *StateWaitingStash) Receive(msg actor.Message) (actor.Actor, error) {
	switch v := msg.(type) {
	case MsgStash:
		a.ch <- 1
		state := StateWaitingUnstash{
			ch: a.ch,
		}
		return &state, errors.Stash
	default:
		return a, fmt.Errorf("StateWaitingStash: don't know what to do with %T", v)
	}
}

func (a *StateWaitingUnstash) Receive(msg actor.Message) (actor.Actor, error) {
	switch v := msg.(type) {
	case MsgUnstash:
		a.ch <- 2
		state := StateProcessingStash{
			ch: a.ch,
		}
		return &state, nil
	default:
		return a, fmt.Errorf("StateWaitingUnstash: don't know what to do with %T", v)
	}
}

func (a *StateProcessingStash) Receive(msg actor.Message) (actor.Actor, error) {
	switch v := msg.(type) {
	case MsgStash:
		a.ch <- 3
		return a, errors.Terminate
	default:
		return a, fmt.Errorf("StateProcessingStash: don't know what to do with %T", v)
	}
}

func TestSystemStashUnstash(t *testing.T) {
	t.Parallel()
	system := New()
	progress := make(chan int, 3)
	pid := system.Spawn(func(system actor.System, pid actor.Pid) (state actor.Actor, limit int) {
		return &StateWaitingStash{ch: progress}, 0
	})
	err := system.Send(pid, MsgStash{})
	require.NoError(t, err)
	err = system.Send(pid, MsgUnstash{})
	require.NoError(t, err)

	require.Equal(t, 1, <-progress)
	require.Equal(t, 2, <-progress)
	require.Equal(t, 3, <-progress)
	system.AwaitTermination()
}

type MailboxFullActor struct {
	ch_received chan struct{}
	ch_waiting chan struct{}
	ch_echo chan int
}

func (a *MailboxFullActor) Receive(msg actor.Message) (actor.Actor, error) {
	switch v := msg.(type) {
	case string:
		// this is necessary to fill the message queue on the calling side
		a.ch_received <- struct{}{}
		<-a.ch_waiting
		return a, nil
	case int:
		a.ch_echo <- v
		if v == 0 {
			return a, errors.Terminate
		}
		return a, nil
	default:
		return a, fmt.Errorf("Don't know what to do with %T", v)
	}
}

func TestSystemMailboxFull(t *testing.T) {
	t.Parallel()
	system := New()
	ch_received := make(chan struct{}, 1)
	ch_waiting := make(chan struct{}, 1)
	ch_echo := make(chan int, 3)
	pid := system.Spawn(func(system actor.System, pid actor.Pid) (state actor.Actor, limit int) {
		limit = 3
		state = &MailboxFullActor{
			ch_received:ch_received,
			ch_waiting: ch_waiting,
			ch_echo: ch_echo,
		}
		return
	})
	err := system.Send(pid, "wait")
	require.NoError(t, err)
	<-ch_received
	err = system.Send(pid, 2)
	require.NoError(t, err)
	err = system.Send(pid, 1)
	require.NoError(t, err)
	err = system.Send(pid, 0)
	require.NoError(t, err)
	err = system.Send(pid, -1)
	require.Equal(t, errors.MailboxFull, err)
	ch_waiting <- struct{}{}

	require.Equal(t, 2, <-ch_echo)
	require.Equal(t, 1, <-ch_echo)
	require.Equal(t, 0, <-ch_echo)
	system.AwaitTermination()
}

type PingPongActor struct {
	system actor.System
	pid actor.Pid
	pong_received chan struct{}
}

type PingMessage struct {
	from actor.Pid
}

type PongMessage struct {
	from actor.Pid
}

type SendPingMessage struct {
	to actor.Pid
	pong_received chan struct{}
}

func (a *PingPongActor) Receive(message actor.Message) (actor.Actor, error) {
	switch v := message.(type) {
	case SendPingMessage:
		a.pong_received = v.pong_received
		_ = a.system.Send(v.to, PingMessage{from: a.pid})
		return a, nil
	case PingMessage:
		_ = a.system.Send(v.from, PongMessage{from: a.pid})
		return a, nil
	case PongMessage:
		a.pong_received <- struct{}{}
		return a, nil
	case TerminateMessage:
		return a, errors.Terminate
	default:
		return a, fmt.Errorf("Don't know what to do with %T", v)
	}
}

func newPingPongActor(system actor.System, pid actor.Pid) (state actor.Actor, limit int) {
	state = &PingPongActor{
		system: system,
		pid: pid,
	}
	return
}

func TestPingPong(t *testing.T) {
	t.Parallel()
	system := New()
	pong_received := make(chan struct{}, 1)
	pid1 := system.Spawn(newPingPongActor)
	pid2 := system.Spawn(newPingPongActor)
	err := system.Send(pid1, SendPingMessage{to: pid2, pong_received: pong_received})
	require.NoError(t, err)
	<-pong_received
	err = system.Send(pid1, TerminateMessage{})
	require.NoError(t, err)
	err = system.Send(pid2, TerminateMessage{})
	require.NoError(t, err)
	system.AwaitTermination()
}
