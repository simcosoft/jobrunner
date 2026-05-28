package jobrunner

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// setup resets global state, starts the scheduler, and registers Stop on cleanup.
func setup(t *testing.T, opts ...int) {
	t.Helper()
	workPermits = nil
	selfConcurrent = false
	Start(opts...)
	t.Cleanup(func() {
		<-Stop().Done()
		workPermits = nil
		selfConcurrent = false
	})
}

// ---------- helpers ----------

// countJob records how many times it ran. Uses pointer receiver so it is
// passed as *countJob; the type name seen by reflect will be empty, which is
// the expected behaviour for pointer types.
type countJob struct {
	n    int32
	done chan struct{}
}

func newCountJob() *countJob {
	return &countJob{done: make(chan struct{}, 1)}
}

func (j *countJob) Run() {
	atomic.AddInt32(&j.n, 1)
	select {
	case j.done <- struct{}{}:
	default:
	}
}

func (j *countJob) Count() int { return int(atomic.LoadInt32(&j.n)) }

// namedJob uses a value receiver so reflect.TypeOf returns the named type "namedJob".
type namedJob struct{ n *int32 }

func (j namedJob) Run() { atomic.AddInt32(j.n, 1) }

// ---------- New ----------

func TestNew_FuncGetsUnnamedName(t *testing.T) {
	j := New(Func(func() {}))
	if j.Name != UNNAMED {
		t.Errorf("expected name %q, got %q", UNNAMED, j.Name)
	}
}

func TestNew_StructGetsTypeName(t *testing.T) {
	var n int32
	j := New(namedJob{n: &n})
	if j.Name != "namedJob" {
		t.Errorf("expected name %q, got %q", "namedJob", j.Name)
	}
}

// ---------- Start ----------

func TestStart_NoOptions(t *testing.T) {
	setup(t)
	if MainCron == nil {
		t.Fatal("MainCron should not be nil after Start")
	}
	if workPermits != nil {
		t.Error("workPermits should be nil when no pool size is given")
	}
}

func TestStart_WithPoolSize(t *testing.T) {
	setup(t, 5)
	if cap(workPermits) != 5 {
		t.Errorf("expected pool capacity 5, got %d", cap(workPermits))
	}
}

func TestStart_DefaultPoolSizeOnZero(t *testing.T) {
	setup(t, 0)
	if cap(workPermits) != DEFAULT_JOB_POOL_SIZE {
		t.Errorf("expected default pool capacity %d, got %d", DEFAULT_JOB_POOL_SIZE, cap(workPermits))
	}
}

func TestStart_SelfConcurrentEnabled(t *testing.T) {
	setup(t, DEFAULT_JOB_POOL_SIZE, 1)
	if !selfConcurrent {
		t.Error("selfConcurrent should be true")
	}
}

func TestStart_SelfConcurrentDisabledOnZero(t *testing.T) {
	setup(t, DEFAULT_JOB_POOL_SIZE, 0)
	if selfConcurrent {
		t.Error("selfConcurrent should be false")
	}
}

func TestStart_ExtraArgsDoNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Start panicked with extra arguments: %v", r)
		}
	}()
	// passing more args than there are option functions must not panic
	setup(t, 10, 1, 99, 42)
}

// ---------- Schedule ----------

func TestSchedule_ValidSpec_ReturnsNoError(t *testing.T) {
	setup(t)
	if err := Schedule("@every 1h", Func(func() {})); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSchedule_ValidSpec_AppearsInEntries(t *testing.T) {
	setup(t)
	_ = Schedule("@every 1h", Func(func() {}))
	if n := len(Entries()); n != 1 {
		t.Errorf("expected 1 entry, got %d", n)
	}
}

func TestSchedule_InvalidSpec_ReturnsError(t *testing.T) {
	setup(t)
	if err := Schedule("not-a-cron-spec", Func(func() {})); err == nil {
		t.Fatal("expected an error for invalid cron spec, got nil")
	}
}

// ---------- Every ----------

func TestEvery_AppearsInEntries(t *testing.T) {
	setup(t)
	Every(1*time.Hour, Func(func() {}))
	if n := len(Entries()); n != 1 {
		t.Errorf("expected 1 entry, got %d", n)
	}
}

// ---------- Now ----------

func TestNow_RunsImmediately(t *testing.T) {
	setup(t)
	job := newCountJob()
	Now(job)
	select {
	case <-job.done:
	case <-time.After(2 * time.Second):
		t.Fatal("job did not run within timeout")
	}
	if job.Count() != 1 {
		t.Errorf("expected count 1, got %d", job.Count())
	}
}

// ---------- In ----------

func TestIn_RunsAfterDelay(t *testing.T) {
	setup(t)
	job := newCountJob()
	In(100*time.Millisecond, job)
	select {
	case <-job.done:
	case <-time.After(2 * time.Second):
		t.Fatal("job did not run within timeout")
	}
	if job.Count() != 1 {
		t.Errorf("expected count 1, got %d", job.Count())
	}
}

// ---------- Stop ----------

func TestStop_ContextCompletesAfterStop(t *testing.T) {
	Start()
	ctx := Stop()
	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("Stop context did not complete within timeout")
	}
}

// ---------- Remove ----------

func TestRemove_JobDisappearsFromEntries(t *testing.T) {
	setup(t)
	if err := Schedule("@every 1h", Func(func() {})); err != nil {
		t.Fatalf("schedule failed: %v", err)
	}
	entries := Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry before Remove, got %d", len(entries))
	}
	Remove(entries[0].ID)
	if n := len(Entries()); n != 0 {
		t.Errorf("expected 0 entries after Remove, got %d", n)
	}
}

// ---------- Panic recovery ----------

func TestJob_PanicDoesNotKillProcess(t *testing.T) {
	setup(t)
	done := make(chan struct{})
	Now(Func(func() {
		defer close(done)
		panic("intentional test panic")
	}))
	select {
	case <-done:
		// process survived – test passes
	case <-time.After(2 * time.Second):
		t.Fatal("panic job did not complete within timeout")
	}
}

// ---------- Work pool ----------

func TestJob_WorkPoolLimitsParallelism(t *testing.T) {
	const poolSize = 3
	setup(t, poolSize)

	var (
		mu     sync.Mutex
		active int
		max    int
		wg     sync.WaitGroup
	)

	const total = 6
	wg.Add(total)

	slowJob := Func(func() {
		mu.Lock()
		active++
		if active > max {
			max = active
		}
		mu.Unlock()

		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		active--
		mu.Unlock()
	})

	for i := 0; i < total; i++ {
		go func() {
			defer wg.Done() // fires after Run() fully returns, including defer <-workPermits
			New(slowJob).Run()
		}()
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("work pool test timed out")
	}

	mu.Lock()
	defer mu.Unlock()
	if max > poolSize {
		t.Errorf("max concurrent jobs was %d, pool size is %d", max, poolSize)
	}
}

// ---------- Self-concurrent guard ----------

func TestJob_SelfConcurrentFalse_PreventsReentrance(t *testing.T) {
	setup(t, DEFAULT_JOB_POOL_SIZE, 0) // selfConcurrent = false

	var (
		active  int32
		overlap bool
		once    sync.Once
	)
	blocking := make(chan struct{})
	first := make(chan struct{})

	// Both goroutines share the same *Job instance, so running.Mutex is shared.
	job := New(Func(func() {
		if atomic.AddInt32(&active, 1) > 1 {
			overlap = true
		}
		once.Do(func() { close(first) }) // signal that first run has started
		<-blocking                       // hold the mutex until released
		atomic.AddInt32(&active, -1)
	}))

	go job.Run()
	<-first // first execution is running and holds the mutex

	done := make(chan struct{})
	go func() {
		job.Run() // must block until first finishes
		close(done)
	}()

	// give the second goroutine time to reach the mutex
	time.Sleep(50 * time.Millisecond)
	close(blocking) // release first execution

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("second run did not complete within timeout")
	}

	if overlap {
		t.Error("two instances of the same job ran concurrently")
	}
}
