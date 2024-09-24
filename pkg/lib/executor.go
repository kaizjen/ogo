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

type Executor struct {
	mutex              sync.Mutex
	queue              []*Task
	nextTaskID         int
	currentlyExecuting map[int]*Task
	completedMutex     sync.Mutex
	completed          map[int]*Task
}

func NewExecutor() Executor {
	return Executor{
		mutex:              sync.Mutex{},
		queue:              []*Task{},
		nextTaskID:         0,
		currentlyExecuting: map[int]*Task{},
		completedMutex:     sync.Mutex{},
		completed:          map[int]*Task{},
	}
}

func floatSecondsToDuration(s float64) time.Duration {
	return time.Duration(s*1000.0) * time.Millisecond
}

type TaskParams struct {
	N   int     `json:"n"`
	N1  float64 `json:"n1"`
	D   float64 `json:"d"`
	I   float64 `json:"I"`
	TTL float64 `json:"TTL"`
}

func (self *Executor) enqueue(params *TaskParams) *Task {
	task := &Task{
		MaxIterations: params.N,
		Delta:         params.D,
		StartValue:    params.N1,
		Interval:      params.I,
		TTL:           params.TTL,
	}
	self.mutex.Lock()

	task.ID = self.nextTaskID
	task.Status = STATUS_IN_QUEUE
	task.EnqueuedAt = time.Now().Unix()
	task.Iteration = 1
	task.CurrentValue = task.StartValue

	self.nextTaskID++
	if len(self.currentlyExecuting) >= N {
		self.queue = append(self.queue, task)
		task.QueueNumber = len(self.queue) - 1
		// Annoying, but this seems to be the best way to
		// do this without deadlocks
		self.mutex.Unlock()

	} else {
		self.mutex.Unlock()
		self.execute(task)
	}
	return task
}

func (self *Executor) execute(task *Task) {
	fmt.Printf("Executing task (id %v)\n", task.ID)
	self.mutex.Lock()
	self.currentlyExecuting[task.ID] = task
	self.mutex.Unlock()

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
			time.Sleep(floatSecondsToDuration(task.Interval))
			task.CurrentValue = task.CurrentValue + task.Delta
		}

		self.mutex.Lock()

		fmt.Printf("Task done! (id %v)\n", task.ID)
		delete(self.currentlyExecuting, task.ID)
		self.ttl(task)

		if len(self.currentlyExecuting) < N {
			if len(self.queue) > 0 {
				qtask := self.queue[0]
				self.queue = self.queue[1:]
				for i := 0; i < len(self.queue); i++ {
					// Shift all QueueNumbers
					self.queue[i].QueueNumber = i
				}
				self.mutex.Unlock()
				self.execute(qtask)
			} else {
				self.mutex.Unlock()
			}
		} else {
			self.mutex.Unlock()
		}
	}()
}

func (self *Executor) ttl(task *Task) {
	task.Status = STATUS_COMPLETED
	task.CompletedAt = time.Now().Unix()

	self.completedMutex.Lock()
	self.completed[task.ID] = task
	self.completedMutex.Unlock()

	go func() {
		time.Sleep(floatSecondsToDuration(task.TTL))

		self.completedMutex.Lock()
		fmt.Printf("Task expired! (id %v)\n", task.ID)
		delete(self.completed, task.ID)
		self.completedMutex.Unlock()
	}()
}

func (self *Executor) list() []*Task {
	list := make([]*Task, 0, len(self.completed)+len(self.currentlyExecuting)+len(self.queue))

	self.completedMutex.Lock()
	for _, value := range self.completed {
		list = append(list, value)
	}
	self.completedMutex.Unlock()

	self.mutex.Lock()
	for _, value := range self.currentlyExecuting {
		list = append(list, value)
	}
	list = append(list, self.queue...)
	self.mutex.Unlock()

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
