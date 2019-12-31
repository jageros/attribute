package attribute

type iAttr interface {
	SetDirty(val bool)
	getRoot() iAttr
	setRoot(root iAttr)
	setParent(parent iAttr, pkey interface{})
	clearParent()
	getParent() iAttr
	getPkey() interface{}
}

type baseAttr struct {
	i      iAttr
	parent iAttr
	pkey   interface{} // key of this item in parent
	//path   []interface{}
	root  iAttr
	dirty bool
}

func (a *baseAttr) Dirty() bool {
	return a.dirty
}

func (a *baseAttr) SetDirty(val bool) {
	a.dirty = val
}

func (a *baseAttr) getRoot() iAttr {
	return a.root
}

func (a *baseAttr) clearParent() {
	a.parent = nil
	a.pkey = nil
	a.i.setRoot(nil)
}

func (a *baseAttr) setParent(parent iAttr, pkey interface{}) {
	a.parent = parent
	a.pkey = pkey
	a.i.setRoot(parent.getRoot())
}

func (a *baseAttr) getParent() iAttr {
	return a.parent
}

func (a *baseAttr) getPkey() interface{} {
	return a.pkey
}
