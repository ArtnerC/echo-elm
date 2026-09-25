// Package workers runs independent jobs — one per CQL library, parity fixture
// or bundle entry — across a bounded number of goroutines.
package workers

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// EnvJobs names the environment variable that overrides the default number of
// workers, for runs whose command line is not easy to change (CI, task files).
const EnvJobs = "ECHO_ELM_JOBS"

// DefaultJobs is the number of workers a run uses when none is asked for: one
// per available CPU, less one left free for the process driving the run and
// whatever else the machine is doing, and never fewer than one. The
// ECHO_ELM_JOBS environment variable, when it holds a positive integer,
// replaces it.
//
// "Available" is GOMAXPROCS, not the machine's CPU count: Go sets it from the
// container's CPU limit, so a pod limited to 4 CPUs on a 64-core node gets 3
// workers, not 63.
func DefaultJobs() int {
	if n, err := strconv.Atoi(strings.TrimSpace(os.Getenv(EnvJobs))); err == nil && n > 0 {
		return n
	}
	return max(1, min(runtime.NumCPU(), runtime.GOMAXPROCS(0))-1)
}

// Resolve returns jobs when it is positive, and DefaultJobs otherwise, so a
// --jobs flag can default to 0 meaning "choose for me".
func Resolve(jobs int) int {
	if jobs > 0 {
		return jobs
	}
	return DefaultJobs()
}

// ForEach calls fn(i) for every i in [0, n), on at most jobs goroutines at once
// (jobs ≤ 0 means DefaultJobs), and returns when every call has returned.
// Calls start in index order but may finish in any order; a caller that needs
// ordered output stores results by index, or uses Ordered.
func ForEach(n, jobs int, fn func(i int)) {
	jobs = min(Resolve(jobs), n)
	if jobs <= 1 {
		for i := range n {
			fn(i)
		}
		return
	}
	next := make(chan int)
	var wg sync.WaitGroup
	for range jobs {
		wg.Go(func() {
			for i := range next {
				fn(i)
			}
		})
	}
	for i := range n {
		next <- i
	}
	close(next)
	wg.Wait()
}

// Ordered is ForEach whose results are handed to emit in index order, as soon
// as each one and all before it are done. A run over hundreds of libraries can
// then report progress as it goes while its output stays byte-for-byte the same
// whatever the number of workers.
//
// emit runs on the calling goroutine, one call at a time, so it may write to
// shared output without locking.
func Ordered[T any](n, jobs int, fn func(i int) T, emit func(i int, result T)) {
	results := make([]T, n)
	done := make([]chan struct{}, n)
	for i := range done {
		done[i] = make(chan struct{})
	}
	finished := make(chan struct{})
	go func() {
		ForEach(n, jobs, func(i int) {
			results[i] = fn(i)
			close(done[i])
		})
		close(finished)
	}()
	for i := range n {
		<-done[i]
		emit(i, results[i])
		var zero T
		results[i] = zero // let the emitted result be collected
	}
	<-finished
}
