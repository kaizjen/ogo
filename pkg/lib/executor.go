package lib

import (
	"fmt"
	"slices"
	"sync"
	"time"
)

var N = 10

type Status int

const (
	STATUS_IN_QUEUE    Status = iota
	STATUS_IN_PROGRESS Status = iota
	STATUS_COMPLETED   Status = iota
)

type Task struct {
	ID     int
	Status Status
	// n
	MaxIterations int
	// d
	Delta        float64
	CurrentValue float64
	StartValue   float64
	Iteration    int
	// I (seconds)
	Interval float64
	// TTL (seconds)
	TTL float64

	// If it's -1, the Task is not in the queue
	QueueNumber int

	// (unix seconds)
	EnqueuedAt int64
	// (unix seconds)
	StartedAt int64
	// (unix seconds)
	CompletedAt int64
}

// Locking queue, currentlyExecuting
var mutex = sync.Mutex{}

// Using mutexes seems way simpler than using a sync channel
// though sync channels may be more robust/performant, idk

var queue = []*Task{}

//            ^ This is a pointer so that the /list endpoint would be simpler,
//              though i have no idea what memory is allocated on the stack
//              and what memory is allocated on the heap in go (and ideally
//              we'd like to avoid heap allocs as much as possible)

var nextTaskID = 0
var currentlyExecuting = map[int]*Task{}

// Locking completed
var completedMutex = sync.Mutex{}
var completed = map[int]*Task{}

func floatSecondsToDuration(s float64) time.Duration {
	return time.Duration(s*1000.0) * time.Millisecond
}

func enqueue(task *Task) {
	fmt.Printf("enq task %v\n", task.ID)
	mutex.Lock()
	fmt.Printf("enq task %v - continue\n", task.ID)

	nextTaskID++
	if len(currentlyExecuting) >= N {
		queue = append(queue, task)
		task.QueueNumber = len(queue) - 1
		// Annoying, but this seems to be the best way to
		// do this without deadlocks
		mutex.Unlock()

	} else {
		mutex.Unlock()
		execute(task)
	}
}

func execute(task *Task) {
	fmt.Printf("Executing task (id %v)\n", task.ID)
	mutex.Lock()
	currentlyExecuting[task.ID] = task
	mutex.Unlock()

	task.Status = STATUS_IN_PROGRESS
	task.StartedAt = time.Now().Unix()
	task.QueueNumber = -1

	// I had to use anon goroutines here because of
	// how testing expects to track tasks, so
	// it's important that if 2 tasks are scheduled at the
	// same time, that they're scheduled in order.
	// However, this wouldn't affect HTTP clients
	// since there's no guaratee about precisely when the request
	// will be sent anyway.
	go func() {
		for ; task.Iteration < task.MaxIterations; task.Iteration++ {
			// if task.Iteration != task.MaxIterations {
			time.Sleep(floatSecondsToDuration(task.Interval))
			// }
			task.CurrentValue = task.CurrentValue + task.Delta
		}

		mutex.Lock()

		fmt.Printf("Task done! (id %v)\n", task.ID)
		delete(currentlyExecuting, task.ID)
		ttl(task)

		if len(currentlyExecuting) < N {
			if len(queue) > 0 {
				qtask := queue[0]
				queue = queue[1:]
				for i := 0; i < len(queue); i++ {
					// Shift all QueueNumbers
					queue[i].QueueNumber = i
				}
				mutex.Unlock()
				execute(qtask)
			} else {
				mutex.Unlock()
			}
		} else {
			mutex.Unlock()
		}
	}()
}

func ttl(task *Task) {
	task.Status = STATUS_COMPLETED
	task.CompletedAt = time.Now().Unix()

	completedMutex.Lock()
	completed[task.ID] = task
	completedMutex.Unlock()

	go func() {
		time.Sleep(floatSecondsToDuration(task.TTL))

		completedMutex.Lock()
		fmt.Printf("Task expired! (id %v)\n", task.ID)
		delete(completed, task.ID)
		completedMutex.Unlock()
	}()
}

func list() []*Task {
	list := make([]*Task, 0, len(completed)+len(currentlyExecuting)+len(queue))

	completedMutex.Lock()
	for _, value := range completed {
		list = append(list, value)
	}
	completedMutex.Unlock()

	mutex.Lock()
	for _, value := range currentlyExecuting {
		list = append(list, value)
	}
	list = append(list, queue...)
	mutex.Unlock()

	slices.SortFunc(list, func(t1, t2 *Task) int {
		if t1.Status == t2.Status {
			return t1.ID - t2.ID
		} else {
			// Sorts tasks like Completed > In progress > In queue
			return int(t2.Status) - int(t1.Status)
		}
	})

	return list
}
