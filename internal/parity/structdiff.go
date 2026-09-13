package parity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// StructuralDiff compares two normalized ELM documents and reports differences
// as paths into the tree, rather than as a line-by-line text diff.
//
// Two properties make the output actually usable for finding translator bugs:
//
// Definitions are aligned by name, not by index. A library that is missing one
// statement, or that emits its statements in a different order, otherwise
// reports every subsequent definition as different — one real difference
// presenting as dozens, and each one attributed to the wrong definition. This is
// how a missing definition turns into a report that `If` became `Or`.
//
// A node that wraps the other side's node is reported as a wrapper, once. When
// echo-elm emits As(Null) where CQF emits Null, the naive comparison says the
// node type differs *and* every field under it is missing, when the real finding
// is a single inserted node.
//
// Diffs are capped: past a certain count the list stops being a report and
// starts being a wall, and the first few are what get acted on anyway.
func StructuralDiff(want, got string) string {
	var w, g any
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		return simpleDiff(want, got)
	}
	if err := json.Unmarshal([]byte(got), &g); err != nil {
		return simpleDiff(want, got)
	}
	d := &differ{max: 40}
	d.compare(w, g, "library")
	if len(d.out) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, line := range d.out {
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	if d.truncated {
		fmt.Fprintf(&sb, "… (%d more)\n", d.suppressed)
	}
	return sb.String()
}

type differ struct {
	out        []string
	max        int
	truncated  bool
	suppressed int
}

func (d *differ) report(format string, args ...any) {
	if len(d.out) >= d.max {
		d.truncated = true
		d.suppressed++
		return
	}
	d.out = append(d.out, fmt.Sprintf(format, args...))
}

// namedContainers are the fields whose array elements are definitions
// identified by name, and so must be aligned by name rather than by position.
var namedContainers = map[string]bool{
	"def": true,
}

func (d *differ) compare(want, got any, path string) {
	if len(d.out) >= d.max {
		d.truncated = true
		return
	}
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			d.report("%s: reference has an object, echo-elm has %s", path, kindOf(got))
			return
		}
		if d.reportWrapper(w, g, path) {
			return
		}
		d.compareObjects(w, g, path)
	case []any:
		g, ok := got.([]any)
		if !ok {
			d.report("%s: reference has a list, echo-elm has %s", path, kindOf(got))
			return
		}
		d.compareLists(w, g, path)
	default:
		if !equalScalar(want, got) {
			d.report("%s: reference %v, echo-elm %v", path, render(want), render(got))
		}
	}
}

func (d *differ) compareObjects(w, g map[string]any, path string) {
	keys := make([]string, 0, len(w)+len(g))
	seen := map[string]bool{}
	for k := range w {
		keys = append(keys, k)
		seen[k] = true
	}
	for k := range g {
		if !seen[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	for _, k := range keys {
		wv, inWant := w[k]
		gv, inGot := g[k]
		switch {
		case inWant && !inGot:
			d.report("%s.%s: only in reference (%s)", path, k, describe(wv))
		case !inWant && inGot:
			d.report("%s.%s: only in echo-elm (%s)", path, k, describe(gv))
		default:
			d.compare(wv, gv, path+"."+k)
		}
	}
}

// compareLists aligns definition arrays by name and everything else by index.
func (d *differ) compareLists(w, g []any, path string) {
	if namedContainers[lastSegment(path)] && allNamed(w) && allNamed(g) {
		d.compareByName(w, g, path)
		return
	}
	if len(w) != len(g) {
		d.report("%s: reference has %d element(s), echo-elm has %d", path, len(w), len(g))
	}
	n := len(w)
	if len(g) < n {
		n = len(g)
	}
	for i := 0; i < n; i++ {
		d.compare(w[i], g[i], fmt.Sprintf("%s[%d]", path, i))
	}
}

// compareByName pairs definitions by their name field, so a missing or reordered
// definition is reported as exactly that instead of shifting every later index.
func (d *differ) compareByName(w, g []any, path string) {
	wByName, wOrder := indexByName(w)
	gByName, gOrder := indexByName(g)

	for _, name := range wOrder {
		gv, ok := gByName[name]
		if !ok {
			d.report("%s: definition %q only in reference", path, name)
			continue
		}
		d.compare(wByName[name], gv, fmt.Sprintf("%s(%s)", path, name))
	}
	for _, name := range gOrder {
		if _, ok := wByName[name]; !ok {
			d.report("%s: definition %q only in echo-elm", path, name)
		}
	}
	// Ordering is meaningful in ELM — a definition must follow what it
	// references — so report it, but as one finding rather than as a difference
	// at every element.
	if shared := sharedOrder(wOrder, gByName); !equalStrings(shared, sharedOrder(gOrder, wByName)) {
		d.report("%s: definitions appear in a different order (reference: %s; echo-elm: %s)",
			path, strings.Join(shared, ", "), strings.Join(sharedOrder(gOrder, wByName), ", "))
	}
}

// reportWrapper detects that one side inserted a node the other does not have,
// and reports the insertion instead of every difference underneath it.
func (d *differ) reportWrapper(w, g map[string]any, path string) bool {
	if inner, field, ok := unwrapsTo(g, w); ok {
		d.report("%s: echo-elm wraps this in %s (via .%s); the operand matches",
			path, typeName(g), field)
		_ = inner
		return true
	}
	if inner, field, ok := unwrapsTo(w, g); ok {
		d.report("%s: reference wraps this in %s (via .%s); the operand matches",
			path, typeName(w), field)
		_ = inner
		return true
	}
	return false
}

// unwrapsTo reports whether outer is a single-child node whose child equals
// target, i.e. outer is target with one node wrapped around it.
func unwrapsTo(outer, target map[string]any) (inner any, field string, ok bool) {
	if typeName(outer) == typeName(target) {
		return nil, "", false
	}
	for _, f := range []string{"operand", "source", "expression"} {
		child, present := outer[f]
		if !present {
			continue
		}
		// A single-element operand array counts: As and the unary operators
		// differ in arity, not in intent.
		if arr, isArr := child.([]any); isArr {
			if len(arr) != 1 {
				continue
			}
			child = arr[0]
		}
		cm, isObj := child.(map[string]any)
		if !isObj {
			continue
		}
		if deepEqual(cm, target) {
			return cm, f, true
		}
	}
	return nil, "", false
}

func indexByName(items []any) (map[string]any, []string) {
	byName := make(map[string]any, len(items))
	order := make([]string, 0, len(items))
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		if _, dup := byName[name]; dup {
			// Overloads share a name; fall back to positional identity for them
			// by suffixing, so they still pair up stably.
			name = fmt.Sprintf("%s#%d", name, len(order))
		}
		byName[name] = it
		order = append(order, name)
	}
	return byName, order
}

// sharedOrder returns the names of order that also exist in other, preserving
// order — so ordering is compared only over definitions both sides have.
func sharedOrder(order []string, other map[string]any) []string {
	out := make([]string, 0, len(order))
	for _, n := range order {
		if _, ok := other[n]; ok {
			out = append(out, n)
		}
	}
	return out
}

func allNamed(items []any) bool {
	if len(items) == 0 {
		return false
	}
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			return false
		}
		if _, ok := m["name"].(string); !ok {
			return false
		}
	}
	return true
}

func typeName(m map[string]any) string {
	if t, ok := m["type"].(string); ok {
		return t
	}
	return "(untyped)"
}

func lastSegment(path string) string {
	if i := strings.LastIndex(path, "."); i >= 0 {
		return path[i+1:]
	}
	return path
}

func kindOf(v any) string {
	switch v.(type) {
	case map[string]any:
		return "an object"
	case []any:
		return "a list"
	case nil:
		return "nothing"
	default:
		return fmt.Sprintf("%v", render(v))
	}
}

func describe(v any) string {
	switch t := v.(type) {
	case map[string]any:
		return typeName(t)
	case []any:
		return fmt.Sprintf("%d element(s)", len(t))
	default:
		return render(v)
	}
}

func render(v any) string {
	if v == nil {
		return "null"
	}
	if s, ok := v.(string); ok {
		return fmt.Sprintf("%q", s)
	}
	return fmt.Sprintf("%v", v)
}

func equalScalar(a, b any) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// deepEqual compares two decoded JSON trees by canonical encoding.
func deepEqual(a, b any) bool {
	ab, err1 := json.Marshal(a)
	bb, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return bytes.Equal(ab, bb)
}
