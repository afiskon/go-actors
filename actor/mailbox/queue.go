package mailbox

import "github.com/afiskon/go-actors/actor"

type queue struct {
	len int
	next *queue
	prev *queue
	payload actor.Message
}

func (q *queue) init() {
	q.next = q
	q.prev = q
	q.payload = nil
	q.len = 0
}

func (q *queue) empty() bool {
	return q.next == q
}

func (q *queue) length() int {
	return q.len
}

func (q *queue) enqueue(payload actor.Message) {
	new :=  &queue{
		payload: payload,
	}

	new.next = q.next
	new.prev = q
	q.next.prev = new
	q.next = new
	q.len++
}

// empty() should be checked before this call
func (q *queue) dequeue() actor.Message {
	msg := q.prev.payload
	old := q.prev

	old.prev.next = q
	q.prev = old.prev
	old.next = nil
	old.prev = nil
	q.len--
	return msg
}

func (q1 *queue) moveFromQueue(q2* queue) {
	if q2.empty() {
		return
	}

	q2.next.prev = q1.prev
	q1.prev.next = q2.next
	q2.prev.next = q1
	q1.prev = q2.prev

	q1.len += q2.len
	q2.init()
}
