package workers

import (
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestDefaultJobsLeavesACPUFree(t *testing.T) {
	t.Setenv(EnvJobs, "")
	want := max(1, min(runtime.NumCPU(), runtime.GOMAXPROCS(0))-1)
	if got := DefaultJobs(); got != want {
		t.Fatalf("DefaultJobs() = %d, want %d", got, want)
	}
}

func TestDefaultJobsEnvOverride(t *testing.T) {
	cases := []struct {
		env  string
		want int // -1: the CPU-derived default
	}{{"3", 3}, {" 12 ", 12}, {"0", -1}, {"-2", -1}, {"many", -1}}
	for _, c := range cases {
		env, want := c.env, c.want
		t.Setenv(EnvJobs, env)
		if want < 0 {
			want = max(1, min(runtime.NumCPU(), runtime.GOMAXPROCS(0))-1)
		}
		if got := DefaultJobs(); got != want {
			t.Errorf("%s=%q: DefaultJobs() = %d, want %d", EnvJobs, env, got, want)
		}
	}
}

func TestResolve(t *testing.T) {
	t.Setenv(EnvJobs, "5")
	if got := Resolve(2); got != 2 {
		t.Errorf("Resolve(2) = %d", got)
	}
	if got := Resolve(0); got != 5 {
		t.Errorf("Resolve(0) = %d, want the default 5", got)
	}
}

func TestForEachRunsEveryIndexOnceWithinTheLimit(t *testing.T) {
	const n, jobs = 200, 4
	var seen [n]atomic.Int32
	var running, peak atomic.Int32
	ForEach(n, jobs, func(i int) {
		r := running.Add(1)
		for {
			p := peak.Load()
			if r <= p || peak.CompareAndSwap(p, r) {
				break
			}
		}
		time.Sleep(100 * time.Microsecond)
		seen[i].Add(1)
		running.Add(-1)
	})
	for i := range seen {
		if c := seen[i].Load(); c != 1 {
			t.Fatalf("index %d ran %d times", i, c)
		}
	}
	if p := peak.Load(); p > jobs {
		t.Fatalf("%d calls ran at once, limit %d", p, jobs)
	}
}

func TestForEachZeroItems(t *testing.T) {
	ForEach(0, 8, func(int) { t.Fatal("called") })
}

func TestOrderedEmitsInIndexOrder(t *testing.T) {
	const n = 100
	var got []int
	Ordered(n, 8, func(i int) int {
		// Later indexes finish first.
		time.Sleep(time.Duration(n-i) * 20 * time.Microsecond)
		return i * i
	}, func(i, sq int) {
		if sq != i*i {
			t.Errorf("result %d = %d", i, sq)
		}
		got = append(got, i)
	})
	for i, v := range got {
		if v != i {
			t.Fatalf("emitted out of order: %v", got)
		}
	}
	if len(got) != n {
		t.Fatalf("emitted %d of %d", len(got), n)
	}
}
