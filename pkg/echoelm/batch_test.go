package echoelm_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/artnerc/echo-elm/internal/resolver"
	"github.com/artnerc/echo-elm/internal/translator"
	"github.com/artnerc/echo-elm/pkg/echoelm"
)

// TestTranslateBatchMatchesTranslate checks that translating libraries in
// parallel with a shared include cache gives byte-for-byte what translating
// each on its own gives. The layered and measure-shapes corpora share
// FHIRHelpers and include each other several levels deep, so the cache is hit
// from many goroutines at once; CI runs this under -race.
func TestTranslateBatchMatchesTranslate(t *testing.T) {
	var inputs []echoelm.BatchInput
	for _, dir := range []string{"layered", "measure-shapes"} {
		root := filepath.Join("..", "..", "test", "corpus", "cqframework", dir)
		files, err := filepath.Glob(filepath.Join(root, "*.cql"))
		if err != nil || len(files) == 0 {
			t.Fatalf("no CQL under %s: %v", root, err)
		}
		src := echoelm.WithLibrarySource(resolver.NewDirSource(root))
		for _, f := range files {
			b, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			inputs = append(inputs, echoelm.BatchInput{Source: b, Name: filepath.Base(f), Options: []echoelm.Option{src}})
		}
	}
	// Twice over, so later copies of a library meet a warm cache while earlier
	// ones are still filling it.
	inputs = append(inputs, inputs...)

	common := []echoelm.Option{echoelm.WithCQFOptions(), func(o *translator.Options) { o.EnableResultTypes = true }}
	marshal := func(r *echoelm.TranslateResult) []byte {
		b, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}

	want := make([][]byte, len(inputs))
	for i, in := range inputs {
		r, err := echoelm.Translate(in.Source, in.Name, append(append([]echoelm.Option{}, common...), in.Options...)...)
		if err != nil {
			t.Fatalf("%s: %v", in.Name, err)
		}
		want[i] = marshal(r)
	}

	for _, jobs := range []int{1, 8} {
		got := echoelm.TranslateBatch(inputs, jobs, common...)
		if len(got) != len(inputs) {
			t.Fatalf("jobs=%d: %d results for %d inputs", jobs, len(got), len(inputs))
		}
		for i, r := range got {
			if r.Err != nil {
				t.Fatalf("jobs=%d %s: %v", jobs, inputs[i].Name, r.Err)
			}
			if !bytes.Equal(marshal(r.Result), want[i]) {
				t.Errorf("jobs=%d %s: batch output differs from Translate", jobs, inputs[i].Name)
			}
		}
	}
}

func TestDefaultJobsEnv(t *testing.T) {
	t.Setenv("ECHO_ELM_JOBS", "3")
	if got := echoelm.DefaultJobs(); got != 3 {
		t.Fatalf("DefaultJobs() = %d with ECHO_ELM_JOBS=3", got)
	}
}
