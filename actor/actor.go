package actor

// Pid is an unique ID of the Actor.
type Pid uint64

// Message represents a message sent between actor.
type Message interface {}

// Mailbox is a queue of messages sent to a given actor.
type Mailbox interface {
	// Enqueue places a new message to the Mailbox. If Mailbox is full
	// method returns errors.MailboxFull. Mailboxes are unbounded unless
	// SetLimit was called with a value of `limit` > 0.
	Enqueue(message Message) error

	// Enqueue places a new message in the front of the messages queue. The
	// message will be Dequeued before any messages placed using Enqueue.
	EnqueueFront(message Message) error

	// Dequeue gets a next message from the Mailbox. If Mailbox is empty,
	// Dequeue blocks until someone places a message to the Mailbox. There
	// should be only one goroutine that uses this method.
	Dequeue() Message

	// Stash places a message to the separate queue of delayed messages.
	// This method should be called from the same single goroutine that calls Dequeue.
	Stash(message Message)

	// Unstash places all previously Stash'ed messages in the front of the message queue.
	// The original order of Stash'ed messages is preserved. This method should be called
	// from the same single goroutine that calls Dequeue.
	Unstash()

	// SetLimit sets the maximum capacity of the Mailbox. This limit doesn't affect
	// stashed and prioritized messages - the amount of these messages is always unlimited.
	SetLimit(limit int)
}

// Actor is an entity that processes messages, sends message to other
// actor and stores some state.
type Actor interface {
	// Receive is called when Actor receives a new Message.
	// Returns a new Actor state and/or error.
	Receive(message Message) (Actor, error)
}

// Constructor is a procedure that creates a new Actor.
// It is called when Actor is created, before receiving any Messages.
// `state` sets the initial actor state. If `limit` is > 0 it sets the
// maximum size of the mailbox (only for regular, not stashed, not
// prioritized messages).
type Constructor func(system System, pid Pid) (state Actor, limit int)

// System a class responsible for creating, scheduling and otherwise
// controlling actor.
type System interface {
	// Send send a Message to the Actor with a given Pid. InvalidPid is returned
	// if actor with given Pid doesn't exists or was terminated.
	Send(pid Pid, message Message) error

	// Spawn creates a new Actor and returns it Pid.
	Spawn(constructor Constructor) Pid

	// AwaitTermination returns when all spawned Actors terminate.
	AwaitTermination()
}