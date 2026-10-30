package respond

import (
	"encoding"
	"encoding/json"
	"net/http"
	"reflect"
)

// Lists is JSON with every nil slice in v written as [] instead of null,
// so a page reading a list of a fresh install (no rows yet) gets an empty
// list. Nil pointers and maps stay null; []byte keeps its own encoding.
func Lists(w http.ResponseWriter, status int, v any) {
	JSON(w, status, EmptyLists(v))
}

// EmptyLists returns a copy of v with every nil slice made empty. It
// leaves v as it is.
func EmptyLists(v any) any {
	if v == nil {
		return nil
	}
	return emptyLists(reflect.ValueOf(v)).Interface()
}

var (
	jsonMarshaler = reflect.TypeFor[json.Marshaler]()
	textMarshaler = reflect.TypeFor[encoding.TextMarshaler]()
)

// ownEncoding reports a type that encodes itself, such as time.Time or
// json.RawMessage: its fields are not walked.
func ownEncoding(t reflect.Type) bool {
	if t.Kind() == reflect.Interface {
		return false
	}
	pt := reflect.PointerTo(t)
	return t.Implements(jsonMarshaler) || t.Implements(textMarshaler) || pt.Implements(jsonMarshaler) || pt.Implements(textMarshaler)
}

func emptyLists(v reflect.Value) reflect.Value {
	t := v.Type()
	if ownEncoding(t) {
		return v
	}
	switch t.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return v
		}
		out := reflect.New(t).Elem()
		out.Set(emptyLists(v.Elem()))
		return out
	case reflect.Pointer:
		if v.IsNil() {
			return v
		}
		out := reflect.New(t.Elem())
		out.Elem().Set(emptyLists(v.Elem()))
		return out
	case reflect.Struct:
		out := reflect.New(t).Elem()
		out.Set(v)
		for i := range t.NumField() {
			if f := out.Field(i); f.CanSet() {
				f.Set(emptyLists(f))
			}
		}
		return out
	case reflect.Slice:
		return emptySlice(v)
	case reflect.Array:
		out := reflect.New(t).Elem()
		for i := range v.Len() {
			out.Index(i).Set(emptyLists(v.Index(i)))
		}
		return out
	case reflect.Map:
		if v.IsNil() {
			return v
		}
		out := reflect.MakeMapWithSize(t, v.Len())
		for it := v.MapRange(); it.Next(); {
			out.SetMapIndex(it.Key(), emptyLists(it.Value()))
		}
		return out
	default:
		return v
	}
}

func emptySlice(v reflect.Value) reflect.Value {
	t := v.Type()
	if t.Elem().Kind() == reflect.Uint8 {
		return v
	}
	out := reflect.MakeSlice(t, v.Len(), v.Len())
	for i := range v.Len() {
		out.Index(i).Set(emptyLists(v.Index(i)))
	}
	return out
}
