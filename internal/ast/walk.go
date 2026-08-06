package ast

import "reflect"

// Walk calls fn for n and every Node reachable from it, in depth-first order.
//
// Traversal is reflective so that it stays correct as node types are added:
// any exported field that is a Node, a slice of Nodes, or a struct/pointer
// containing them is followed. Nil nodes are skipped and fn is never called
// with one.
func Walk(n Node, fn func(Node)) {
	if n == nil {
		return
	}
	seen := make(map[uintptr]bool)
	walkValue(reflect.ValueOf(n), fn, seen)
}

var nodeType = reflect.TypeOf((*Node)(nil)).Elem()

func walkValue(v reflect.Value, fn func(Node), seen map[uintptr]bool) {
	if !v.IsValid() {
		return
	}

	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return
		}
		walkValue(v.Elem(), fn, seen)
		return

	case reflect.Pointer:
		if v.IsNil() {
			return
		}
		// Guard against cycles and repeated shared subtrees.
		if ptr := v.Pointer(); seen[ptr] {
			return
		} else {
			seen[ptr] = true
		}
		if v.Type().Implements(nodeType) {
			if node, ok := v.Interface().(Node); ok {
				fn(node)
			}
		}
		walkValue(v.Elem(), fn, seen)
		return

	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			walkValue(v.Index(i), fn, seen)
		}
		return

	case reflect.Struct:
		// A non-pointer struct can itself be a Node (e.g. QueryRelationship).
		if v.CanInterface() && v.Type().Implements(nodeType) {
			if node, ok := v.Interface().(Node); ok {
				fn(node)
			}
		}
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if t.Field(i).PkgPath != "" {
				continue // unexported
			}
			walkValue(v.Field(i), fn, seen)
		}
		return
	}
}
