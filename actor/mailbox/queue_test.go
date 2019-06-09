package mailbox

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewQueueIsEmtpy(t *testing.T) {
	t.Parallel()
	q := queue{}
	q.init()
	require.True(t, q.empty())
	require.Equal(t, 0, q.length())
}

func TestQueueEnqueueDequeue(t *testing.T) {
	t.Parallel()
	q := queue{}
	q.init()
	inMsg := struct{ payload string}{ payload: "hello" }
	require.True(t, q.empty())
	require.Equal(t, 0, q.length())
	q.enqueue(inMsg)
	require.False(t, q.empty())
	require.Equal(t, 1, q.length())
	outMsg := q.dequeue()
	require.True(t, q.empty())
	require.Equal(t, inMsg, outMsg)
}

func TestQueueOrdering(t *testing.T) {
	t.Parallel()
	q := queue{}
	q.init()
	msg1 := struct{ payload string }{ payload: "msg1" }
	msg2 := struct{ payload string }{ payload: "msg2" }
	msg3 := struct{ payload string }{ payload: "msg3" }
	q.enqueue(msg1)
	q.enqueue(msg2)
	q.enqueue(msg3)
	require.Equal(t, msg1, q.dequeue())
	require.Equal(t, msg2, q.dequeue())
	require.Equal(t, msg3, q.dequeue())
	require.True(t, q.empty())
}

func TestQueueMoveFromQueue(t *testing.T) {
	t.Parallel()
	q1_messages := []struct{ payload string }{
		{ payload: "q1-msg1" },
		{ payload: "q1-msg2" },
		{ payload: "q1-msg3" },
	}
	q2_messages := []struct{ payload string }{
		{ payload: "q2-msg1" },
		{ payload: "q2-msg2" },
		{ payload: "q2-msg3" },
	}

	for q1len := 0; q1len <= 3; q1len++ {
		for q2len := 0; q2len <= 3; q2len++ {
			q1 := queue{}
			q1.init()
			q2 := queue{}
			q2.init()

			for i := 0; i < q1len; i++ {
				q1.enqueue(q1_messages[i])
			}

			for i := 0; i < q2len; i++ {
				q2.enqueue(q2_messages[i])
			}

			require.Equal(t, q1len, q1.length())
			require.Equal(t, q2len, q2.length())
			q1.moveFromQueue(&q2)
			require.Equal(t, q1len+q2len, q1.length())
			require.Equal(t, 0, q2.length())

			for i := 0; i < q2len; i++ {
				msg := q1.dequeue()
				require.Equal(t, q2_messages[i], msg)
			}

			for i := 0; i < q1len; i++ {
				msg := q1.dequeue()
				require.Equal(t, q1_messages[i], msg)
			}
		}
	}
}
