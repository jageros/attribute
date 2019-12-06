package mongo

import (
	"attribute/timer"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"log"
	"time"
)

type attrData struct {
	attrID interface{}
	data   map[string]interface{}
}

func (ad *attrData) GetAttrID() interface{} {
	return ad.attrID
}

func (ad *attrData) GetData() map[string]interface{} {
	return ad.data
}

type MongoDBEngine struct {
	session    *mgo.Session
	database   string
	pingTicker *timer.Timer
}

func OpenMongoDB(addr, dbname, user, passowrd string) (*MongoDBEngine, error) {
	log.Printf("Connecting MongoDB %s ...", addr)
	session, err := mgo.Dial("mongodb://" + addr + "/")
	if err != nil {
		return nil, err
	}

	db := session.DB(dbname)
	if user != "" {
		if err = db.Login(user, passowrd); err != nil {
			return nil, err
		}
	}

	session.SetMode(mgo.Strong, true)

	pingTicker := timer.AddTicker(10*time.Second, func() {
		session.Ping()
	})

	return &MongoDBEngine{
		session:    session,
		database:   dbname,
		pingTicker: pingTicker,
	}, nil
}

func (e *MongoDBEngine) Write(attrName string, attrID interface{}, data map[string]interface{}) error {
	col := e.getCollection(attrName)
	_, err := col.UpsertId(attrID, bson.M{
		"data": data,
	})
	col.Insert()
	col.Database.Session.Close()
	return err
}

func (e *MongoDBEngine) Insert(attrName string, attrID interface{}, data map[string]interface{}) error {
	col := e.getCollection(attrName)
	err := col.Insert(bson.M{"_id": attrID, "data": data})
	col.Database.Session.Close()
	return err
}

func (e *MongoDBEngine) Query(attrName string) (func() (attrID interface{}, data map[string]interface{}, hasMore bool), error) {
	col := e.session.DB(e.database).C(attrName)
	iter := col.Find(nil).Iter()
	return func() (attrID interface{}, data map[string]interface{}, hasMore bool) {
		var doc bson.M
		if iter.Next(&doc) {
			attrID = doc["_id"]
			data = e.convertM2Map(doc["data"].(bson.M))
			hasMore = true
		}
		return
	}, nil
}

func (e *MongoDBEngine) Read(attrName string, attrID interface{}) (map[string]interface{}, error) {
	col := e.getCollection(attrName)
	q := col.FindId(attrID)
	var doc bson.M
	err := q.One(&doc)
	col.Database.Session.Close()

	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return e.convertM2Map(doc["data"].(bson.M)), nil
}

func (e *MongoDBEngine) convertM2Map(m bson.M) map[string]interface{} {
	ma := map[string]interface{}(m)
	e.convertM2MapInMap(ma)
	return ma
}

func (e *MongoDBEngine) convertM2MapInMap(m map[string]interface{}) {
	for k, v := range m {
		switch im := v.(type) {
		case bson.M:
			m[k] = e.convertM2Map(im)
		case map[string]interface{}:
			e.convertM2MapInMap(im)
		case []interface{}:
			e.convertM2MapInList(im)
		}
	}
}

func (e *MongoDBEngine) convertM2MapInList(l []interface{}) {
	for i, v := range l {
		switch im := v.(type) {
		case bson.M:
			l[i] = e.convertM2Map(im)
		case map[string]interface{}:
			e.convertM2MapInMap(im)
		case []interface{}:
			e.convertM2MapInList(im)
		}
	}
}

func (e *MongoDBEngine) getCollection(attrName string) *mgo.Collection {
	ses := e.session.Copy()
	return ses.DB(e.database).C(attrName)
}

func (e *MongoDBEngine) Exists(attrName string, attrID interface{}) (bool, error) {
	col := e.getCollection(attrName)
	query := col.FindId(attrID)
	var doc bson.M
	err := query.One(&doc)
	col.Database.Session.Close()

	if err == nil {
		// doc found
		return true, nil
	} else if err == mgo.ErrNotFound {
		return false, nil
	} else {
		return false, err
	}
}

func (e *MongoDBEngine) Close() {
	e.session.Close()
	if e.pingTicker != nil {
		e.pingTicker.Cancel()
	}
}

func (e *MongoDBEngine) IsEOF(err error) bool {
	return err == io.EOF || err == io.ErrUnexpectedEOF
}

func (e *MongoDBEngine) Del(attrName string, attrID interface{}) error {
	col := e.getCollection(attrName)
	err := col.RemoveId(attrID)
	col.Database.Session.Close()
	return err
}

func (e *MongoDBEngine) ReadAll(attrName string) ([]interface {
	GetAttrID() interface{}
	GetData() map[string]interface{}
}, error) {

	col := e.getCollection(attrName)
	q := col.Find(bson.M{})
	var docs []bson.M
	err := q.All(&docs)
	col.Database.Session.Close()

	if err != nil {
		if err == mgo.ErrNotFound {
			return []interface {
				GetAttrID() interface{}
				GetData() map[string]interface{}
			}{}, nil
		}
		return nil, err
	}

	var datas []interface {
		GetAttrID() interface{}
		GetData() map[string]interface{}
	}
	for _, doc := range docs {
		datas = append(datas, &attrData{
			attrID: doc["_id"],
			data:   e.convertM2Map(doc["data"].(bson.M)),
		})
	}

	return datas, nil
}
