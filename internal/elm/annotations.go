package elm

import (
	"encoding/json"
	"reflect"
)

var rawMessageType = reflect.TypeOf(json.RawMessage(nil))

// emptyAnnotation is the container CQF writes on an Element with no annotations.
var emptyAnnotation = json.RawMessage("[]")

// FillEmptyAnnotations gives every Element in the tree that carries no
// annotation the empty container CQF writes on it.
//
// In echo-elm's model only ELM Elements have an Annotation field — definitions,
// clauses, expressions, type specifiers, TupleElementDefinition — while the
// non-Element structures (TupleElement, InstanceElement) have none. So "set
// every nil Annotation field" is exactly CQF's rule, and it reaches the nodes
// that are synthesized rather than translated: inferred result-type specifiers,
// aggregate clauses, tuple element definitions. Filling them site by site had
// already missed eight categories.
//
// Call it only in CQF-compatible mode; the modern output omits empty containers.
func FillEmptyAnnotations(root any) {
	fillAnnotations(reflect.ValueOf(root), map[uintptr]bool{})
}

func fillAnnotations(v reflect.Value, seen map[uintptr]bool) {
	switch v.Kind() {
	case reflect.Interface:
		if !v.IsNil() {
			fillAnnotations(v.Elem(), seen)
		}
	case reflect.Pointer:
		if v.IsNil() {
			return
		}
		if p := v.Pointer(); seen[p] {
			return
		} else {
			seen[p] = true
		}
		fillAnnotations(v.Elem(), seen)
	case reflect.Slice:
		if v.Type() == rawMessageType {
			return
		}
		for i := 0; i < v.Len(); i++ {
			fillAnnotations(v.Index(i), seen)
		}
	case reflect.Struct:
		if f := v.FieldByName("Annotation"); f.IsValid() && f.Type() == rawMessageType && f.IsNil() && f.CanSet() {
			f.Set(reflect.ValueOf(emptyAnnotation))
		}
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				fillAnnotations(v.Field(i), seen)
			}
		}
	}
}
