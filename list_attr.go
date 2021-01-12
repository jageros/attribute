package attribute

import (
	"fmt"
	"strings"
)

type ListAttr struct {
	baseAttr
	items []interface{}
}

func NewListAttr() *ListAttr {
	a := &ListAttr{
		items: make([]interface{}, 0),
	}
	a.root = a
	a.i = a
	return a
}

func (a *ListAttr) setRoot(root iAttr) {
	a.root = root
	for _, sa := range a.items {
		if _sa, ok := sa.(iAttr); ok {
			_sa.setRoot(root)
		}
	}
}

func (a *ListAttr) String() string {
	var sb strings.Builder
	sb.WriteString("ListAttr{")
	isFirstField := true
	for _, v := range a.items {
		if !isFirstField {
			sb.WriteString(", ")
		}

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

func (a *ListAttr) Size() int {
	return len(a.items)
}

func (a *ListAttr) set(index int, val interface{}) {
	a.items[index] = val
	if sa, ok := val.(iAttr); ok {
		parent := sa.getParent()
		pkey := sa.getPkey()
		if (parent != nil && parent != a) || (pkey != nil && pkey != index) {
			panic(fmt.Sprintf("attr reused in index %d", index))
		}

		sa.setParent(a, index)
	}
	a.root.SetDirty(true)
}

func (a *ListAttr) Get(index int) interface{} {
	return a.items[index]
}

func (a *ListAttr) GetInt(index int) int {
	return a.Get(index).(int)
}

func (a *ListAttr) GetInt32(index int) int32 {
	val := a.Get(index)
	if v, ok := val.(int32); ok {
		return v
	} else {
		return int32(val.(int))
	}
}

func (a *ListAttr) GetUInt32(index int) uint32 {
	val := a.Get(index)
	if v, ok := val.(uint32); ok {
		return v
	} else {
		return uint32(val.(int))
	}
}

func (a *ListAttr) GetUInt64(index int) uint64 {
	val := a.Get(index)
	if v, ok := val.(uint64); ok {
		return v
	} else {
		return uint64(val.(int64))
	}
}

func (a *ListAttr) GetFloat32(index int) float32 {
	val := a.Get(index)
	if v, ok := val.(float32); ok {
		return v
	} else {
		return float32(val.(float64))
	}
}

func (a *ListAttr) GetStr(index int) string {
	return a.Get(index).(string)
}

func (a *ListAttr) GetBool(index int) bool {
	return a.Get(index).(bool)
}

func (a *ListAttr) GetListAttr(index int) *ListAttr {
	val := a.Get(index)
	return val.(*ListAttr)
}

func (a *ListAttr) GetMapAttr(index int) *MapAttr {
	val := a.Get(index)
	return val.(*MapAttr)
}

func (a *ListAttr) AppendInt(v int) {
	a.Append(v)
}

func (a *ListAttr) AppendUInt64(v uint64) {
	a.Append(v)
}

func (a *ListAttr) AppendInt32(v int32) {
	a.Append(v)
}

func (a *ListAttr) AppendUInt32(v uint32) {
	a.Append(v)
}

func (a *ListAttr) AppendFloat32(v float32) {
	a.Append(v)
}

func (a *ListAttr) AppendBool(v bool) {
	a.Append(v)
}

func (a *ListAttr) AppendStr(v string) {
	a.Append(v)
}

func (a *ListAttr) AppendMapAttr(attr *MapAttr) {
	a.Append(attr)
}

func (a *ListAttr) AppendListAttr(attr *ListAttr) {
	a.Append(attr)
}

func (a *ListAttr) pop() interface{} {
	size := len(a.items)
	val := a.items[size-1]
	a.items = a.items[:size-1]

	if sa, ok := val.(iAttr); ok {
		sa.clearParent()
	}
	a.root.SetDirty(true)

	return val
}

func (a *ListAttr) PopInt() int {
	return a.pop().(int)
}

func (a *ListAttr) PopFloat32() float32 {
	val := a.pop()
	if v, ok := val.(float32); ok {
		return v
	} else {
		return float32(val.(float64))
	}
}

func (a *ListAttr) PopBool() bool {
	return a.pop().(bool)
}

func (a *ListAttr) PopStr() string {
	return a.pop().(string)
}

func (a *ListAttr) PopListAttr() *ListAttr {
	return a.pop().(*ListAttr)
}

func (a *ListAttr) PopMapAttr() *MapAttr {
	return a.pop().(*MapAttr)
}

func (a *ListAttr) DelInt(val int) int {
	for i, _ := range a.items {
		if a.GetInt(i) == val {
			a.DelByIndex(i)
			return i
		}
	}
	return -1
}

func (a *ListAttr) DelUint32(val uint32) int {
	for i, _ := range a.items {
		if a.GetUInt32(i) == val {
			a.DelByIndex(i)
			return i
		}
	}
	return -1
}

func (a *ListAttr) DelStr(val string) int {
	for i, _ := range a.items {
		if a.GetStr(i) == val {
			a.DelByIndex(i)
			return i
		}
	}
	return -1
}

func (a *ListAttr) DelMapAttr(val *MapAttr) int {
	for i, _ := range a.items {
		if a.GetMapAttr(i) == val {
			a.DelByIndex(i)
			return i
		}
	}
	return -1
}

func (a *ListAttr) DelByIndex(index int) {
	if index >= 0 && index < len(a.items) {
		a.items = append(a.items[:index], a.items[index+1:]...)
		if a.root != nil {
			a.root.SetDirty(true)
		}
	}
}

// del [beginIdx, endIdx)
// beginIdx >= 0
// endIdx <= list len
func (a *ListAttr) DelBySection(beginIdx, endIdx int) {
	if beginIdx < 0 {
		beginIdx = 0
	}
	if endIdx <= beginIdx {
		return
	}

	size := a.Size()
	if endIdx >= size {
		a.items = a.items[:beginIdx]
	} else {
		a.items = append(a.items[:beginIdx], a.items[endIdx:]...)
	}
	if a.root != nil {
		a.root.SetDirty(true)
	}
}

func (a *ListAttr) Append(val interface{}) {
	a.items = append(a.items, val)
	index := len(a.items) - 1

	if sa, ok := val.(iAttr); ok {
		parent := sa.getParent()
		pkey := sa.getPkey()
		if (parent != nil && parent != a) || (pkey != nil && pkey != index) {
			panic(fmt.Sprintf("attr reused in index %d", index))
		}

		sa.setParent(a, index)
	}
	a.root.SetDirty(true)
}

func (a *ListAttr) SetInt(index int, v int) {
	if a.GetInt(index) == v {
		return
	}
	a.set(index, v)
}

func (a *ListAttr) SetInt32(index int, v int32) {
	if a.GetInt32(index) == v {
		return
	}
	a.set(index, v)
}

func (a *ListAttr) SetUInt32(index int, v uint32) {
	if a.GetUInt32(index) == v {
		return
	}
	a.set(index, v)
}

func (a *ListAttr) SetUInt64(index int, v uint64) {
	if a.GetUInt64(index) == v {
		return
	}
	a.set(index, v)
}

func (a *ListAttr) SetFloat32(index int, v float32) {
	if a.GetFloat32(index) == v {
		return
	}
	a.set(index, v)
}

func (a *ListAttr) SetBool(index int, v bool) {
	if a.GetBool(index) == v {
		return
	}
	a.set(index, v)
}

func (a *ListAttr) SetStr(index int, v string) {
	if a.GetStr(index) == v {
		return
	}
	a.set(index, v)
}

func (a *ListAttr) SetMapAttr(index int, attr *MapAttr) {
	a.set(index, attr)
}

func (a *ListAttr) SetListAttr(index int, attr *ListAttr) {
	a.set(index, attr)
}

func (a *ListAttr) ToList() []interface{} {
	l := make([]interface{}, len(a.items))

	for i, v := range a.items {
		switch a := v.(type) {
		case *MapAttr:
			l[i] = a.ToMap()
		case *ListAttr:
			l[i] = a.ToList()
		default:
			l[i] = v
		}
	}
	return l
}

func (a *ListAttr) AssignList(l []interface{}) {
	for _, v := range l {
		switch iv := v.(type) {
		case map[string]interface{}:
			ia := NewMapAttr()
			ia.AssignMap(iv)
			a.Append(ia)
		case []interface{}:
			ia := NewListAttr()
			ia.AssignList(iv)
			a.Append(ia)
		default:
			a.Append(v)
		}
	}
}

func (a *ListAttr) ForEachIndex(f func(index int) bool) {
	for i, _ := range a.items {
		if !f(i) {
			break
		}
	}
}

func (a *ListAttr) ForOrderEachIndex(order bool, num int, f func(index int) bool) {
	l := len(a.items)
	if num <= 0 || num > l {
		num = l
	}
	if order {
		for i := 0; i < num; i++ {
			if !f(i) {
				break
			}
		}
	} else {
		for i := l - 1; i >= l-num; i-- {
			if !f(i) {
				break
			}
		}
	}
}

func (a *ListAttr) ForIntervalIndex(startIndex, endIndex int, f func(index int) bool) {
	l := len(a.items)
	if endIndex > l {
		endIndex = l
	}
	for i := startIndex; i < endIndex; i++ {
		if !f(i) {
			break
		}
	}
}
