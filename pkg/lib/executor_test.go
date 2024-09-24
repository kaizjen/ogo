package lib

import (
	"runtime/debug"
	"testing"
	"time"
)

// We add epsilon to every time.Sleep() so that we don't fail the test
// because of desync between goroutines.
// (this never affects HTTP clients because they won't get millisecond
// percision anyways, due to how the protocol works)
const epsilon = 1.0 / 100.0

func FailTest(t *testing.T, fmt string, other ...any) {
	debug.PrintStack()
	t.Fatalf("\033[31m"+fmt+"\033[0m", other...)
}

func testEnqueue(exec *Executor, n int, d, n1, i, ttl float64) *Task {
	task := TaskParams{
		N:   n,
		D:   d,
		N1:  n1,
		I:   i,
		TTL: ttl,
	}

	return exec.enqueue(&task)
}

func testTaskIsInQueue(t *testing.T, task *Task, indexInQueue int) {
	if task.Status != STATUS_IN_QUEUE {
		FailTest(t, "incorrect status (want in queue, got %v)", task.Status)
	}
	if task.QueueNumber != indexInQueue {
		FailTest(t, "incorrect queue index (want %v, got %v)", indexInQueue, task.QueueNumber)
	}
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

	// Unix seconds lack precision, so we add one second
	// instead of an epsilon so that we know for sure that
	// the needed amount of time had passed
	timePassed := (taskTTLAt - float64(time.Now().Unix())) + 1

	time.Sleep(floatSecondsToDuration(timePassed))

	return timePassed
}

func testTaskDestruction(t *testing.T, exec *Executor, task *Task) {
	exec.completedMutex.Lock()
	defer exec.completedMutex.Unlock()

	if _, ok := exec.completed[task.ID]; ok {
		FailTest(t, "task %v still exists after it should've been destroyed", task.ID)
	}
}

func TestMultiple(t *testing.T) {
	// hardcoded
	N = 2

	exec := NewExecutor()

	interval := float64(1.2)
	n := int(3)

	ttl := 3.2

	task0 := testEnqueue(&exec, n, 1.5, 2.2, interval, ttl)
	task1 := testEnqueue(&exec, n, 2.5, 3.2, interval, ttl)
	task2 := testEnqueue(&exec, n, 3.5, 4.2, interval, ttl)

	for iteration := range n - 1 {
		testTaskIsInProgress(t, task0, iteration+1)
		testTaskIsInProgress(t, task1, iteration+1)
		testTaskIsInQueue(t, task2, 0)

		time.Sleep(floatSecondsToDuration(interval + epsilon))
	}

	testTaskCompletion(t, task0)
	testTaskCompletion(t, task1)

	for iteration := range n - 1 {
		testTaskIsInProgress(t, task2, iteration+1)
		time.Sleep(floatSecondsToDuration(interval + epsilon))
	}

	testTaskCompletion(t, task2)

	// Testing TTL:
	time.Sleep(floatSecondsToDuration(ttl + epsilon))

	testTaskDestruction(t, &exec, task0)
	testTaskDestruction(t, &exec, task1)
	testTaskDestruction(t, &exec, task2)
}
