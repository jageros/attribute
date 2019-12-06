package db

import (
	"fmt"
	"github.com/jageros/attribute/db/mongo"
	"github.com/jageros/attribute/evq"
	"github.com/jageros/attribute/opmon"
	"github.com/xiaonanln/go-xnsyncutil/xnsyncutil"
	"log"
	"sync"
	"time"
)

var clients []IDbClient

type iDbEngine interface {
	Read(attrName string, attrID interface{}) (map[string]interface{}, error)

	ReadAll(attrName string) ([]interface {
		GetAttrID() interface{}
		GetData() map[string]interface{}
	}, error)

	Query(attrName string) (func() (attrID interface{}, data map[string]interface{}, hasMore bool), error)

	Write(attrName string, attrID interface{}, data map[string]interface{}) error
	Insert(attrName string, attrID interface{}, data map[string]interface{}) error
	Del(attrName string, attrID interface{}) error
	Exists(attrName string, attrID interface{}) (bool, error)
	Close()
	IsEOF(err error) bool
}

type IDbConfig interface {
	GetType() string
	GetAddr() string
	GetDB() string
	GetUser() string
	GetPassword() string
}

type IDbClient interface {
	getConfig() IDbConfig
	Insert(attrName string, attrID interface{}, data map[string]interface{}) error
	Save(attrName string, attrID interface{}, data map[string]interface{}, needReply bool) error
	Del(attrName string, attrID interface{}, needReply bool) error
	Load(attrName string, attrID interface{}, isSync ...interface{}) (map[string]interface{}, error)
	Exists(attrName string, attrID interface{}) (bool, error)
	ForEach(attrName string, callback func(attrID interface{}, data map[string]interface{}))
	shutdown()

	LoadAll(attrName string) ([]interface {
		GetAttrID() interface{}
		GetData() map[string]interface{}
	}, error)
}

type dbClient struct {
	cfg                  IDbConfig
	dbEngine             iDbEngine
	operationQueue       *xnsyncutil.SyncQueue
	recentWarnedQueueLen int
	shutdownOnce         sync.Once
	shutdownNotify       chan struct{}
}

func GetOrNewDbClient(cfg IDbConfig) IDbClient {
	for _, cli := range clients {
		cfg2 := cli.getConfig()
		if cfg2.GetType() == cfg.GetType() && cfg2.GetAddr() == cfg.GetAddr() && cfg2.GetDB() == cfg.GetDB() {
			return cli
		}
	}

	cli := &dbClient{
		cfg:            cfg,
		operationQueue: xnsyncutil.NewSyncQueue(),
		shutdownNotify: make(chan struct{}),
		shutdownOnce:   sync.Once{},
	}

	err := cli.assureDBEngineReady()
	if err != nil {
		log.Fatalf("db engine %s is not ready: %s", cfg, err)
	}

	clients = append(clients, cli)
	go cli.dbRoutine()
	return cli
}

func (c *dbClient) getConfig() IDbConfig {
	return c.cfg
}

func (c *dbClient) assureDBEngineReady() (err error) {
	if c.dbEngine != nil {
		return
	}

	if c.cfg.GetType() == "mongodb" {
		c.dbEngine, err = mongo.OpenMongoDB(c.cfg.GetAddr(), c.cfg.GetDB(), c.cfg.GetUser(), c.cfg.GetPassword())
	} else {
		panic(fmt.Sprintf("unknown db type: %s", c.cfg.GetType()))
	}

	return
}

func (c *dbClient) Insert(attrName string, attrID interface{}, data map[string]interface{}) error {
	req := &insertRequest{
		attrName: attrName,
		attrID:   attrID,
		data:     data,
		c:        make(chan error, 1),
	}

	c.operationQueue.Push(req)
	c.checkOperationQueueLen()

	var err error
	evq.Await(func() {
		err = <-req.c
	})
	return err
}

func (c *dbClient) Save(attrName string, attrID interface{}, data map[string]interface{}, needReply bool) error {
	req := &saveRequest{
		attrName: attrName,
		attrID:   attrID,
		data:     data,
	}
	if needReply {
		req.c = make(chan error, 1)
	}

	c.operationQueue.Push(req)
	c.checkOperationQueueLen()

	if needReply {
		var err error
		evq.Await(func() {
			err = <-req.c
		})
		return err
	} else {
		return nil
	}
}

func (c *dbClient) Del(attrName string, attrID interface{}, needReply bool) error {
	req := &delRequest{
		attrName: attrName,
		attrID:   attrID,
	}
	if needReply {
		req.c = make(chan error, 1)
	}

	c.operationQueue.Push(req)
	c.checkOperationQueueLen()

	if needReply {
		var err error
		evq.Await(func() {
			err = <-req.c
		})
		return err
	} else {
		return nil
	}
}

func (c *dbClient) Load(attrName string, attrID interface{}, isSync ...interface{}) (map[string]interface{}, error) {
	req := &loadRequest{
		attrName: attrName,
		attrID:   attrID,
		c:        make(chan *loadResult, 1),
	}

	c.operationQueue.Push(req)
	c.checkOperationQueueLen()

	var result *loadResult
	if len(isSync) > 0 && isSync[0].(bool) {
		result = <-req.c
	} else {
		evq.Await(func() {
			result = <-req.c
		})
	}
	return result.data, result.err
}

func (c *dbClient) Exists(attrName string, attrID interface{}) (bool, error) {
	req := &existsRequest{
		attrName: attrName,
		attrID:   attrID,
		c:        make(chan *existsResult, 1),
	}

	c.operationQueue.Push(req)
	c.checkOperationQueueLen()

	var result *existsResult
	evq.Await(func() {
		result = <-req.c
	})
	return result.exists, result.err
}

func (c *dbClient) LoadAll(attrName string) ([]interface {
	GetAttrID() interface{}
	GetData() map[string]interface{}
}, error) {

	req := &loadAllRequest{
		attrName: attrName,
		c:        make(chan *loadAllResult, 1),
	}

	c.operationQueue.Push(req)
	c.checkOperationQueueLen()

	var result *loadAllResult
	evq.Await(func() {
		result = <-req.c
	})
	return result.datas, result.err
}

func (c *dbClient) ForEach(attrName string, callback func(attrID interface{}, data map[string]interface{})) {
	req := &forEachRequest{attrName: attrName, iter: nil, c: make(chan *forEachResult, 1),}

	for true {
		c.operationQueue.Push(req)
		c.checkOperationQueueLen()

		var result *forEachResult
		evq.Await(func() {
			result = <-req.c
		})

		if result.err != nil {
			return
		}

		if !result.hasMore {
			return
		}
		callback(result.attrID, result.data)
	}
}

func (c *dbClient) checkOperationQueueLen() {
	qlen := c.operationQueue.Len()
	if qlen > 100 && qlen%100 == 0 && c.recentWarnedQueueLen != qlen {
		log.Printf("db %s operation queue length = %d", c.cfg, qlen)
		c.recentWarnedQueueLen = qlen
	}
}

func (c *dbClient) shutdown() {
	c.shutdownOnce.Do(func() {
		var waitTime time.Duration
		for c.operationQueue.Len() > 0 {
			if waitTime > 10*time.Second {
				log.Printf("db %s Shutdown timeout, left op %d", c.cfg, c.operationQueue.Len())
				break
			}
			t := 100 * time.Millisecond
			waitTime += t
			time.Sleep(t)
		}

		c.operationQueue.Close()
		<-c.shutdownNotify
	})
}

func (c *dbClient) dbRoutine() {
	defer func() {
		err := recover()
		if err != nil {
			log.Printf("db %s routine paniced: %s", c.cfg, err)
		} else {
			c.dbEngine.Close()
			c.dbEngine = nil
			close(c.shutdownNotify)
		}
	}()

	for {
		err := c.assureDBEngineReady()
		if err != nil {
			log.Fatalf("db %s engine is not ready: %s", c.cfg, err)
			time.Sleep(time.Second)
			continue
		}

		if c.dbEngine == nil {
			log.Fatalf("db %s engine is nil", c.cfg)
		}

		req := c.operationQueue.Pop()
		if req == nil {
			break
		}

		req2, ok := req.(iDbRequest)
		if !ok {
			log.Printf("db: unknown operation: %v", req)
			continue
		}

		op := opmon.StartOperation(fmt.Sprintf("db:%s", req2.name()))

		err = req2.execute(c.dbEngine)
		if err != nil {
			log.Fatalf("db: %s %s failed: %s", c.cfg, req2.name(), err)

			if err != nil && c.dbEngine.IsEOF(err) {
				c.dbEngine.Close()
				c.dbEngine = nil
			}
		}

		op.Finish(100 * time.Millisecond)
	}
}

type iDbRequest interface {
	name() string
	execute(engine iDbEngine) error
}

type saveRequest struct {
	attrName string
	attrID   interface{}
	data     map[string]interface{}
	c        chan error
}

func (r *saveRequest) name() string {
	return "save"
}

func (r *saveRequest) execute(engine iDbEngine) error {
	err := engine.Write(r.attrName, r.attrID, r.data)
	if r.c != nil {
		r.c <- err
	}
	return err
}

type delRequest struct {
	attrName string
	attrID   interface{}
	c        chan error
}

func (r *delRequest) name() string {
	return "del"
}

func (r *delRequest) execute(engine iDbEngine) error {
	err := engine.Del(r.attrName, r.attrID)
	if r.c != nil {
		r.c <- err
	}
	return err
}

type loadRequest struct {
	attrName string
	attrID   interface{}
	c        chan *loadResult
}

type loadResult struct {
	data map[string]interface{}
	err  error
}

func (r *loadRequest) name() string {
	return "load"
}

func (r *loadRequest) execute(engine iDbEngine) error {
	data, err := engine.Read(r.attrName, r.attrID)
	if err != nil {
		data = nil
	}

	if r.c != nil {
		r.c <- &loadResult{
			data: data,
			err:  err,
		}
	}
	return err
}

type existsRequest struct {
	attrName string
	attrID   interface{}
	c        chan *existsResult
}

type existsResult struct {
	exists bool
	err    error
}

func (r *existsRequest) name() string {
	return "exists"
}

func (r *existsRequest) execute(engine iDbEngine) error {
	exists, err := engine.Exists(r.attrName, r.attrID)
	if r.c != nil {
		r.c <- &existsResult{
			exists: exists,
			err:    err,
		}
	}
	return err
}

type loadAllRequest struct {
	attrName string
	c        chan *loadAllResult
}

type loadAllResult struct {
	datas []interface {
		GetAttrID() interface{}
		GetData() map[string]interface{}
	}
	err error
}

func (r *loadAllRequest) name() string {
	return "loadAll"
}

func (r *loadAllRequest) execute(engine iDbEngine) error {
	datas, err := engine.ReadAll(r.attrName)
	if err != nil {
		datas = nil
	}

	if r.c != nil {
		r.c <- &loadAllResult{
			datas: datas,
			err:   err,
		}
	}
	return err
}

type forEachRequest struct {
	attrName string
	iter     func() (attrID interface{}, data map[string]interface{}, hasMore bool)
	c        chan *forEachResult
}

type forEachResult struct {
	attrID  interface{}
	data    map[string]interface{}
	hasMore bool
	err     error
}

func (r *forEachRequest) name() string {
	return "forEach"
}

func (r *forEachRequest) execute(engine iDbEngine) error {
	var err error
	var attrID interface{}
	var data map[string]interface{}
	var hasMore bool
	if r.iter != nil {
		attrID, data, hasMore = r.iter()
	} else {
		r.iter, err = engine.Query(r.attrName)
		if err == nil {
			attrID, data, hasMore = r.iter()
		}
	}

	r.c <- &forEachResult{
		attrID:  attrID,
		data:    data,
		hasMore: hasMore,
		err:     err,
	}
	return err
}

type insertRequest struct {
	attrName string
	attrID   interface{}
	data     map[string]interface{}
	c        chan error
}

func (r *insertRequest) name() string {
	return "insert"
}

func (r *insertRequest) execute(engine iDbEngine) error {
	err := engine.Insert(r.attrName, r.attrID, r.data)
	if r.c != nil {
		r.c <- err
	}
	return err
}

func Shutdown() {
	for _, c := range clients {
		c.shutdown()
	}
	clients = []IDbClient{}
}
