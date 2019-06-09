package mailbox

import (
	"github.com/stretchr/testify/require"
	"github.com/afiskon/go-actors/actor/errors"
	"testing"
	"time"
)

func TestMailboxEnqueueDequeue(t *testing.T) {
	t.Parallel()
	mb := New()
	err := mb.Enqueue("msg1")
	require.NoError(t, err)
	msg := mb.Dequeue()
	require.Equal(t, "msg1", msg)
}

func TestMailboxOrdering(t *testing.T) {
	t.Parallel()
	mb := New()
	err := mb.Enqueue("msg1")
	require.NoError(t, err)
	err = mb.Enqueue("msg2")
	require.NoError(t, err)
	err = mb.Enqueue("msg3")
	require.NoError(t, err)
	msg := mb.Dequeue()
	require.Equal(t, "msg1", msg)
	msg = mb.Dequeue()
	require.Equal(t, "msg2", msg)
	msg = mb.Dequeue()
	require.Equal(t, "msg3", msg)
}

func TestMailboxEnqueueFront(t *testing.T) {
	t.Parallel()
	mb := New()
	err := mb.Enqueue("msg1")
	require.NoError(t, err)
	err = mb.Enqueue("msg2")
	require.NoError(t, err)
	err = mb.EnqueueFront("important!")
	require.NoError(t, err)
	msg := mb.Dequeue()
	require.Equal(t, "important!", msg)
	msg = mb.Dequeue()
	require.Equal(t, "msg1", msg)
	msg = mb.Dequeue()
	require.Equal(t, "msg2", msg)
}

func TestMailboxStashUnstash(t *testing.T) {
	t.Parallel()
	mb := New()
	err := mb.Enqueue("msg1")
	require.NoError(t, err)
	err = mb.Enqueue("msg2")
	require.NoError(t, err)
	err = mb.Enqueue("msg3")
	require.NoError(t, err)

	msg := mb.Dequeue()
	require.Equal(t, "msg1", msg)
	mb.Stash(msg)

	msg = mb.Dequeue()
	require.Equal(t, "msg2", msg)
	mb.Unstash()

	msg = mb.Dequeue()
	require.Equal(t, "msg1", msg)
	msg = mb.Dequeue()
	require.Equal(t, "msg3", msg)
}

func TestMailboxEnqueueWaiting(t *testing.T) {
	t.Parallel()
	mb := New()
	result := make(chan string, 1)

	go func() {
		result <- mb.Dequeue().(string)
	}()

	for {
		impl := mb.(*mailbox)
		impl.lock.Lock()
		waiting := impl.dequeueIsWaiting
		impl.lock.Unlock()

		if waiting {
			break
		}

		time.Sleep(10 * time.Millisecond)
	}

	err := mb.Enqueue("hello!")
	require.NoError(t, err)
	require.Equal(t, "hello!", <-result)
}

func TestMailboxEnqueueFrontWaiting(t *testing.T) {
	t.Parallel()
	mb := New()
	result := make(chan string, 1)

	go func() {
		result <- mb.Dequeue().(string)
	}()

	for {
		impl := mb.(*mailbox)
		impl.lock.Lock()
		waiting := impl.dequeueIsWaiting
		impl.lock.Unlock()

		if waiting {
			break
		}

		time.Sleep(10 * time.Millisecond)
	}

	err := mb.EnqueueFront("hello!")
	require.NoError(t, err)
	require.Equal(t, "hello!", <-result)
}

func TestMailboxSetLimit(t *testing.T) {
	t.Parallel()
	mb := New()
	mb.SetLimit(2)
	err := mb.Enqueue("msg1")
	require.NoError(t, err)
	err = mb.Enqueue("msg2")
	require.NoError(t, err)
	err = mb.Enqueue("msg3")
	require.Equal(t, errors.MailboxFull, err)

	err = mb.EnqueueFront("important!")
	require.NoError(t, err)

	msg := mb.Dequeue()
	require.Equal(t, "important!", msg)
	msg = mb.Dequeue()
	require.Equal(t, "msg1", msg)
	msg = mb.Dequeue()
	require.Equal(t, "msg2", msg)
}
