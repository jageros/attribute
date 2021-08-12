package attribute

import (
	"errors"
	"github.com/jageros/db"
	"github.com/jageros/group"
)

var NotExistsErr = errors.New("NotExistsErr")

var DbConfigCreator db.IDbConfig

type AttrMgr struct {
	*MapAttr
	dbClient db.IDbClient
	name     string
	id       interface{}
}

func NewAttrMgr(name string, id interface{}) *AttrMgr {
	return &AttrMgr{
		name:     name,
		id:       id,
		MapAttr:  NewMapAttr(),
		dbClient: db.GetOrNewDbClient(DbConfigCreator),
	}
}

func (a *AttrMgr) Load(isSync ...interface{}) error {
	if data, err := a.dbClient.Load(a.name, a.id, isSync...); err != nil {
		return err
	} else {
		if data == nil {
			return NotExistsErr
		}
		a.AssignMap(data)
		a.SetDirty(false)
		return nil
	}
}

func (a *AttrMgr) Copy(id interface{}, isSync ...interface{}) error {
	if data, err := a.dbClient.Load(a.name, id, isSync...); err != nil {
		return err
	} else {
		if data == nil {
			return NotExistsErr
		}
		a.AssignMap(data)
		a.SetDirty(false)
		return nil
	}
}

func (a *AttrMgr) Save(needReply bool) error {
	if a.Dirty() {
		data := a.ToMap()
		a.SetDirty(false)
		return a.dbClient.Save(a.name, a.id, data, needReply)
	} else {
		return nil
	}
}

func (a *AttrMgr) Insert() error {
	data := a.ToMap()
	return a.dbClient.Insert(a.name, a.id, data)
}

func (a *AttrMgr) Delete(needReply bool) error {
	return a.dbClient.Del(a.name, a.id, needReply)
}

func (a *AttrMgr) Exists() (bool, error) {
	return a.dbClient.Exists(a.name, a.id)
}

func (a *AttrMgr) GetAttrID() interface{} {
	return a.id
}

func LoadAll(attrName string) ([]*AttrMgr, error) {
	datas, err := db.GetOrNewDbClient(DbConfigCreator).LoadAll(attrName)
	if err != nil {
		return nil, err
	}

	var attrs []*AttrMgr
	for _, data := range datas {
		a := NewAttrMgr(attrName, data.GetAttrID())
		a.AssignMap(data.GetData())
		a.SetDirty(false)
		attrs = append(attrs, a)
	}
	return attrs, nil
}

func ForEach(attrName string, callback func(*AttrMgr)) {
	db.GetOrNewDbClient(DbConfigCreator).ForEach(attrName, func(attrID interface{}, data map[string]interface{}) {
		a := NewAttrMgr(attrName, attrID)
		a.AssignMap(data)
		a.SetDirty(false)
		callback(a)
	})
}

func Initialize(g *group.Group, opfs ...db.OpFn) {
	db.Initialize(g)
	DbConfigCreator = db.IDbConfigCreator(opfs...)
}
