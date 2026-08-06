package elm

import "reflect"

// SetResultType records an expression's resolved type on the node, as CQF does
// under --result-types. A named type goes in resultTypeName; a structured one
// (list, interval, tuple, choice) goes in resultTypeSpecifier. Passing both
// empty clears the node.
//
// Every expression node carries the two fields, so this is set reflectively
// rather than through a 48-case type switch.
func SetResultType(e Expression, name string, spec TypeSpecifier) {
	if e == nil {
		return
	}
	v := reflect.ValueOf(e)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return
	}
	if f := v.FieldByName("ResultTypeName"); f.IsValid() && f.CanSet() && f.Kind() == reflect.String {
		f.SetString(name)
	}
	if f := v.FieldByName("ResultTypeSpecifier"); f.IsValid() && f.CanSet() {
		if spec == nil {
			f.Set(reflect.Zero(f.Type()))
		} else if sv := reflect.ValueOf(spec); sv.Type().AssignableTo(f.Type()) {
			f.Set(sv)
		}
	}
}

// HasResultType reports whether a result type has already been recorded.
func HasResultType(e Expression) bool {
	if e == nil {
		return false
	}
	v := reflect.ValueOf(e)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return false
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return false
	}
	if f := v.FieldByName("ResultTypeName"); f.IsValid() && f.Kind() == reflect.String && f.String() != "" {
		return true
	}
	if f := v.FieldByName("ResultTypeSpecifier"); f.IsValid() && !f.IsZero() {
		return true
	}
	return false
}

// GetResultType returns the result type recorded on a node, if any.
func GetResultType(e Expression) (string, TypeSpecifier) {
	if e == nil {
		return "", nil
	}
	v := reflect.ValueOf(e)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return "", nil
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return "", nil
	}
	var name string
	var spec TypeSpecifier
	if f := v.FieldByName("ResultTypeName"); f.IsValid() && f.Kind() == reflect.String {
		name = f.String()
	}
	if f := v.FieldByName("ResultTypeSpecifier"); f.IsValid() && !f.IsZero() {
		spec, _ = f.Interface().(TypeSpecifier)
	}
	return name, spec
}
