package attribute

import (
	"errors"
	"github.com/jageros/attribute/internal/pkg/db"
	"github.com/jageros/attribute/internal/pkg/evq"
	"github.com/jageros/attribute/internal/pkg/timer"
	"time"
)

var NotExistsErr = errors.New("NotExistsErr")

// args[0]: isGlobal(bool), args[1]: region(uint32)
var DbConfigCreator func(args ...interface{}) db.IDbConfig

type AttrMgr struct {
	*MapAttr
	dbClient db.IDbClient
	name     string
	id       interface{}
}

func NewAttrMgr(name string, id interface{}, args ...interface{}) *AttrMgr {
	return &AttrMgr{
		name:     name,
		id:       id,
		MapAttr:  NewMapAttr(),
		dbClient: db.GetOrNewDbClient(DbConfigCreator(args...)),
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

func LoadAll(attrName string, args ...interface{}) ([]*AttrMgr, error) {
	datas, err := db.GetOrNewDbClient(DbConfigCreator(args...)).LoadAll(attrName)
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

func ForEach(attrName string, callback func(*AttrMgr), args ...interface{}) {
	db.GetOrNewDbClient(DbConfigCreator(args...)).ForEach(attrName, func(attrID interface{}, data map[string]interface{}) {
		a := NewAttrMgr(attrName, attrID)
		a.AssignMap(data)
		a.SetDirty(false)
		callback(a)
	})
}

func Start(iDb db.IDbConfig) {
	DbConfigCreator = func(args ...interface{}) db.IDbConfig {
		return iDb
	}
	timer.StartTicks(time.Second)
}

func Stop() {
	timer.StartTicks(time.Millisecond * 500)
	evq.Stop()
	db.Shutdown()
}
