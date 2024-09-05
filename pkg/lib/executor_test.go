package lib

import (
	"sync"
	"testing"
	"time"
)

// We add epsilon to every time.Sleep() so that we don't fail the test
// because of desync between goroutines.
// (this never affects HTTP clients because they won't get millisecond
// percision anyways, due to how the protocol works)
const epsilon = 1.0 / 100.0

func FailTest(t *testing.T, fmt string, other ...any) {
	t.Fatalf("\033[31m"+fmt+"\033[0m", other...)
}

func testEnqueue(n int, d, n1, i, ttl float64) *Task {
	task := Task{
		ID:            nextTaskID,
		MaxIterations: n,
		Delta:         d,
		CurrentValue:  n1,
		StartValue:    n1,
		Iteration:     1,
		Interval:      i,
		TTL:           ttl,
		Status:        STATUS_IN_QUEUE,
		EnqueuedAt:    time.Now().Unix(),
	}

	enqueue(&task)

	return &task
}

func waitUntilOutOfQueue(t *testing.T, task *Task) {
	for task.Status == STATUS_IN_QUEUE {

	}
	t.Logf("Task %d out of queue", task.ID)
}

// Waits for a task to complete (accounting for iterations already done on the task)
func waitForTask(t *testing.T, task *Task) float64 {
	if task.Status == STATUS_IN_QUEUE {
		FailTest(t, "when calling waitForTask(), task %v should not be in queue", task.ID)
	}

	timeCompletion := float64(task.StartedAt) + (float64(task.MaxIterations-1) * task.Interval)
	timePassed := timeCompletion - float64(time.Now().Unix()) + epsilon

	time.Sleep(floatSecondsToDuration(timePassed))
	t.Logf("Wait for task %d", task.ID)
	return timePassed
}

func testTaskIsInQueue(t *testing.T, task *Task, indexInQueue int) {
	if task.Status != STATUS_IN_QUEUE {
		FailTest(t, "incorrect status (want in queue, got %v)", task.Status)
	}
	if task.QueueNumber != indexInQueue {
		FailTest(t, "incorrect queue index (want %v, got %v)", indexInQueue, task.QueueNumber)
	}
}

func testTaskIsInProgressAfterTime(t *testing.T, task *Task) {
	if task.Status == STATUS_IN_QUEUE {
		FailTest(t, "expected task %v to not be in queue", task.ID)
	}

	timePassed := float64(time.Now().Unix()-task.StartedAt) + epsilon
	intervals := int((timePassed / task.Interval))
	testTaskIsInProgress(t, task, 1+intervals)
}

func testTaskIsInProgress(t *testing.T, task *Task, iteration int) {
	value := task.StartValue + (float64(iteration-1) * task.Delta)

	if task.Status != STATUS_IN_PROGRESS {
		FailTest(t, "incorrect status (want in progress, got %v) - id %v", task.Status, task.ID)
	}
	if iteration != task.Iteration {
		FailTest(t, "incorrect iteration count (want %v, got %v) - id %v", iteration, task.Iteration, task.ID)
	}
	if value != task.CurrentValue {
		FailTest(t, "incorrect value (want %v, got %v) - id %v", value, task.CurrentValue, task.ID)
	}
}

func testTaskCompletion(t *testing.T, task *Task) {
	finalValue := task.StartValue + (float64(task.MaxIterations-1) * task.Delta)

	if task.Status != STATUS_COMPLETED {
		FailTest(t, "incorrect status (want completed, got %v) - id %v", task.Status, task.ID)
	}
	if task.Iteration != task.MaxIterations {
		FailTest(t, "incorrect final iteration count (want %v, got %v) - id %v", task.MaxIterations, task.Iteration, task.ID)
	}
	if task.CurrentValue != task.CurrentValue {
		FailTest(t, "incorrect final value (want %v, got %v) - id %v", finalValue, task.CurrentValue, task.ID)
	}
}

func waitForTaskDestruction(t *testing.T, task *Task) float64 {
	if task.Status != STATUS_COMPLETED {
		FailTest(t, "task is not completed yet, id %v\n", task.ID)
	}

	taskTTLAt := float64(task.CompletedAt) + task.TTL

	// HACK: instead of the usual epsilon,
	// we just add a second since sometimes
	// the mutex will be locked and the ttl()
	// function will not be able get a hold of it
	// in time.

	// maybe should disable these types of tests in general
	// since they're so unreliable

	// the other tests don't seem to have problems like this
	timePassed := (taskTTLAt - float64(time.Now().Unix())) + epsilon

	time.Sleep(floatSecondsToDuration(timePassed))

	return timePassed
}

func testTaskDestruction(t *testing.T, task *Task) {
	completedMutex.Lock()
	defer completedMutex.Unlock()

	if _, ok := completed[task.ID]; ok {
		FailTest(t, "task %v still exists after it should've been destroyed", task.ID)
	}
}

func TestMultiple(t *testing.T) {
	t.Logf("TestSingleEnqueue:")
	N = 2
	nextTaskID = 0

	interval := float64(2)
	n := 2

	// It's important that `n` and `i` are increasing
	// because tests rely on the order of completion
	// task0 -> task1 -> task2 -> task3
	task0 := testEnqueue(n, 1.5, 2.2, interval, 5.3)
	task1 := testEnqueue(n+1, 2.5, 3.2, interval+1, 5.3)
	task2 := testEnqueue(n+2, 3.5, 4.2, interval+2, 5.3)
	task3 := testEnqueue(n+3, 4.5, 5.2, interval+3, 5.3)

	testTaskIsInProgressAfterTime(t, task0)
	testTaskIsInProgressAfterTime(t, task1)
	testTaskIsInQueue(t, task2, 0)
	testTaskIsInQueue(t, task3, 1)

	var wg sync.WaitGroup

	// TODO: make it a loop

	wg.Add(1)
	go func() {
		defer wg.Done()
		waitForTask(t, task0)
		testTaskCompletion(t, task0)
		testTaskIsInProgressAfterTime(t, task1)
		testTaskIsInProgressAfterTime(t, task2)
		testTaskIsInQueue(t, task3, 0)

		waitForTaskDestruction(t, task0)
		testTaskDestruction(t, task0)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		waitForTask(t, task1)
		testTaskCompletion(t, task1)
		testTaskIsInProgressAfterTime(t, task2)
		testTaskIsInProgressAfterTime(t, task3)

		waitForTaskDestruction(t, task1)
		testTaskDestruction(t, task1)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		waitUntilOutOfQueue(t, task2)
		waitForTask(t, task2)
		testTaskCompletion(t, task2)
		testTaskIsInProgressAfterTime(t, task3)

		waitForTaskDestruction(t, task2)
		testTaskDestruction(t, task2)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		waitUntilOutOfQueue(t, task3)
		waitForTask(t, task3)
		testTaskCompletion(t, task3)
		waitForTaskDestruction(t, task3)
		testTaskDestruction(t, task3)
	}()

	wg.Wait()
}
