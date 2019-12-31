package attribute

import (
	"fmt"
	"strings"
)

type MapAttr struct {
	baseAttr
	attrs map[string]interface{}
}

func NewMapAttr() *MapAttr {
	a := &MapAttr{
		attrs: make(map[string]interface{}),
	}
	a.root = a
	a.i = a
	return a
}

func (a *MapAttr) setRoot(root iAttr) {
	a.root = root
	for _, sa := range a.attrs {
		if _sa, ok := sa.(iAttr); ok {
			_sa.setRoot(root)
		}
	}
}

func (a *MapAttr) String() string {
	var sb strings.Builder
	sb.WriteString("MapAttr{")
	isFirstField := true
	for k, v := range a.attrs {
		if !isFirstField {
			sb.WriteString(", ")
		}

		fmt.Fprintf(&sb, "%#s", k)
		sb.WriteString(": ")
		switch a := v.(type) {
		case *MapAttr:
			sb.WriteString(a.String())
		case *ListAttr:
			sb.WriteString(a.String())
		default:
			fmt.Fprintf(&sb, "%#s", v)
		}
		isFirstField = false
	}
	sb.WriteString("}")
	return sb.String()
}

func (a *MapAttr) Size() int {
	return len(a.attrs)
}

func (a *MapAttr) HasKey(key string) bool {
	_, ok := a.attrs[key]
	return ok
}

func (a *MapAttr) Keys() []string {
	keys := make([]string, 0, len(a.attrs))
	for k, _ := range a.attrs {
		keys = append(keys, k)
	}
	return keys
}

func (a *MapAttr) ForEachKey(f func(key string)) {
	for k, _ := range a.attrs {
		f(k)
	}
}

func (a *MapAttr) Set(key string, val interface{}) {
	a.attrs[key] = val
	if sa, ok := val.(iAttr); ok {
		parent := sa.getParent()
		pkey := sa.getPkey()
		if (parent != nil && parent != a) || (pkey != nil && pkey != key) {
			panic(fmt.Sprintf("attr reused in key %s", key))
		}

		sa.setParent(a, key)
	}
	a.root.SetDirty(true)
}

func (a *MapAttr) SetInt(key string, v int) {
	if a.GetInt(key) == v {
		return
	}
	a.Set(key, v)
}

func (a *MapAttr) SetInt32(key string, v int32) {
	if a.GetInt32(key) == v {
		return
	}
	a.Set(key, v)
}

func (a *MapAttr) SetUInt32(key string, v uint32) {
	if a.GetUInt32(key) == v {
		return
	}
	a.Set(key, v)
}

func (a *MapAttr) SetUInt64(key string, v uint64) {
	if a.GetUInt64(key) == v {
		return
	}
	a.Set(key, v)
}

func (a *MapAttr) SetInt64(key string, v int64) {
	if a.GetInt64(key) == v {
		return
	}
	a.Set(key, v)
}

func (a *MapAttr) SetFloat(key string, v float32) {
	if a.GetFloat32(key) == v {
		return
	}
	a.Set(key, v)
}

func (a *MapAttr) SetFloat64(key string, v float64) {
	if a.GetFloat64(key) == v {
		return
	}
	a.Set(key, v)
}

func (a *MapAttr) SetBool(key string, v bool) {
	if a.GetBool(key) == v {
		return
	}
	a.Set(key, v)
}

func (a *MapAttr) SetStr(key string, v string) {
	if a.GetStr(key) == v {
		return
	}
	a.Set(key, v)
}

func (a *MapAttr) SetMapAttr(key string, attr *MapAttr) {
	a.Set(key, attr)
}

func (a *MapAttr) SetListAttr(key string, attr *ListAttr) {
	a.Set(key, attr)
}

/*
func (a *MapAttr) getPathFromOwner() []interface{} {
	if a.path == nil {
		a.path = a._getPathFromOwner()
	}
	return a.path
}

func (a *MapAttr) _getPathFromOwner() []interface{} {
	if a.parent == nil {
		return nil
	}

	path := make([]interface{}, 0, 4)
	path = append(path, a.pkey)
	return getPathFromOwner(a.parent, path)
}
*/

func (a *MapAttr) Get(key string) interface{} {
	return a.attrs[key]
}

func (a *MapAttr) GetInt(key string) int {
	val := a.Get(key)
	if val == nil {
		return 0
	} else {
		return val.(int)
	}
}

func (a *MapAttr) GetUInt64(key string) uint64 {
	val := a.Get(key)
	if val == nil {
		return 0
	} else {
		if v, ok := val.(uint64); ok {
			return v
		} else {
			return uint64(val.(int64))
		}
	}
}

func (a *MapAttr) GetInt64(key string) int64 {
	val := a.Get(key)
	if val == nil {
		return 0
	} else {
		if v, ok := val.(int64); ok {
			return v
		} else {
			return int64(val.(int))
		}
	}
}

func (a *MapAttr) GetUInt32(key string) uint32 {
	val := a.Get(key)
	if val == nil {
		return 0
	} else {
		if v, ok := val.(uint32); ok {
			return v
		} else {
			return uint32(val.(int))
		}
	}
}

func (a *MapAttr) GetInt32(key string) int32 {
	val := a.Get(key)
	if val == nil {
		return 0
	} else {
		if v, ok := val.(int32); ok {
			return v
		} else {
			return int32(val.(int))
		}
	}
}

func (a *MapAttr) GetStr(key string) string {
	val := a.Get(key)
	if val == nil {
		return ""
	} else {
		return val.(string)
	}
}

func (a *MapAttr) GetFloat32(key string) float32 {
	val := a.Get(key)
	if val == nil {
		return 0
	} else {
		if v, ok := val.(float32); ok {
			return v
		} else {
			return float32(val.(float64))
		}
	}
}

func (a *MapAttr) GetFloat64(key string) float64 {
	val := a.Get(key)
	if val == nil {
		return 0
	} else {
		if v, ok := val.(float64); ok {
			return v
		} else {
			return val.(float64)
		}
	}
}

func (a *MapAttr) GetBool(key string) bool {
	val := a.Get(key)
	if val == nil {
		return false
	} else {
		return val.(bool)
	}
}

func (a *MapAttr) GetMapAttr(key string) *MapAttr {
	val := a.Get(key)
	if val == nil {
		return nil
	} else {
		return val.(*MapAttr)
	}
}

func (a *MapAttr) GetListAttr(key string) *ListAttr {
	val := a.Get(key)
	if val == nil {
		return nil
	} else {
		return val.(*ListAttr)
	}
}

func (a *MapAttr) Del(key string) {
	val, ok := a.attrs[key]
	if !ok {
		return
	}

	delete(a.attrs, key)
	if sa, ok := val.(*MapAttr); ok {
		sa.clearParent()
	} else if sa, ok := val.(*ListAttr); ok {
		sa.clearParent()
	}
	a.root.SetDirty(true)
}

func (a *MapAttr) ToMap() map[string]interface{} {
	doc := make(map[string]interface{})
	for k, v := range a.attrs {
		switch a := v.(type) {
		case *MapAttr:
			doc[k] = a.ToMap()
		case *ListAttr:
			doc[k] = a.ToList()
		default:
			doc[k] = v
		}
	}
	return doc
}

func (a *MapAttr) ToMapWithFilter(filter func(string) bool) map[string]interface{} {
	doc := map[string]interface{}{}
	for k, v := range a.attrs {
		if !filter(k) {
			continue
		}

		switch a := v.(type) {
		case *MapAttr:
			doc[k] = a.ToMap()
		case *ListAttr:
			doc[k] = a.ToList()
		default:
			doc[k] = v
		}
	}
	return doc
}

func (a *MapAttr) AssignMap(doc map[string]interface{}) {
	for k, v := range doc {
		switch iv := v.(type) {
		case map[string]interface{}:
			ia := NewMapAttr()
			ia.AssignMap(iv)
			a.Set(k, ia)
		case []interface{}:
			ia := NewListAttr()
			ia.AssignList(iv)
			a.Set(k, ia)
		default:
			a.Set(k, v)
		}
	}
}

func (a *MapAttr) AssignMapWithFilter(doc map[string]interface{}, filter func(string) bool) {
	for k, v := range doc {
		if !filter(k) {
			continue
		}

		switch iv := v.(type) {
		case map[string]interface{}:
			ia := NewMapAttr()
			ia.AssignMap(iv)
			a.Set(k, ia)
		case []interface{}:
			ia := NewListAttr()
			ia.AssignList(iv)
			a.Set(k, ia)
		default:
			a.Set(k, v)
		}
	}
}
