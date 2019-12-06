package evq

import (
	"attribute/consts"
	"attribute/utils"
	"fmt"
	"gopkg.in/eapache/queue.v1"
	"log"
	"strconv"
	"sync"
	"time"
)

/*
func GoID() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}
	return id
}
*/

var mainEvScheduler = &eventScheduler{
	eventHandlers: make(map[int][]*handler),
	eventMap:      make(map[uint64]*handler),
	maxCoCount:    10000,
}

func init() {
	mainEvScheduler.start()
}

type IEvent interface {
	GetEventId() int
}

type CommonEvent struct {
	id   int
	data []interface{}
}

func (ce *CommonEvent) GetEventId() int {
	return ce.id
}

func (ce *CommonEvent) String() string {
	return strconv.Itoa(ce.id)
}

func (ce *CommonEvent) GetData() []interface{} {
	return ce.data
}

func NewCommonEvent(evenID int, data ...interface{}) *CommonEvent {
	e := &CommonEvent{
		data: data,
	}
	e.id = evenID
	return e
}

// implement IEvent
type Coroutine struct {
	id     int
	goid   int
	signal chan interface{}
}

func (c *Coroutine) GetEventId() int {
	return consts.COROUTINE_EVENT
}

func (c *Coroutine) String() string {
	return "COROUTINE_EVENT"
}

func (c *Coroutine) Yield() interface{} {
	return <-c.signal
}

func (c *Coroutine) Resume(data interface{}) {
	c.signal <- data
}

func (c *Coroutine) Exit() {
	close(c.signal)
}

func newCoroutine() *Coroutine {
	return &Coroutine{
		signal: make(chan interface{}),
	}
}

type callLaterEvent struct {
	f func()
}

func (ce *callLaterEvent) GetEventId() int {
	return consts.CALLATER_EVT
}

func newCallLaterEvent(f func()) IEvent {
	return &callLaterEvent{
		f: f,
	}
}

type closeEvent struct {
}

func (ce *closeEvent) GetEventId() int {
	return consts.ClOSE_EVQ_EVT
}

type handler struct {
	id   uint64
	evId int
	cb   func(IEvent)
}

type eventQueue struct {
	evList *queue.Queue
	coList *queue.Queue
	guard  sync.Mutex
	cond   *sync.Cond
}

func (eq *eventQueue) String() string {
	return fmt.Sprintf("evList=%s, coList=%s", eq.evList, eq.coList)
}

func (eq *eventQueue) push(event IEvent) {
	if event == nil {
		return
	}

	eq.guard.Lock()
	if event.GetEventId() != consts.COROUTINE_EVENT {
		eq.evList.Add(event)
	} else {
		eq.coList.Add(event)
	}
	eq.cond.Signal()
	eq.guard.Unlock()
}

func (eq *eventQueue) pop() IEvent {
	eq.guard.Lock()

	for eq.empty() {
		eq.cond.Wait()
	}

	var ev IEvent = nil
	if eq.coList.Length() > 0 {
		ev = eq.coList.Remove().(IEvent)
	} else {
		ev = eq.evList.Remove().(IEvent)
	}

	eq.guard.Unlock()
	return ev
}

func (eq *eventQueue) empty() bool {
	return eq.evList.Length() == 0 && eq.coList.Length() == 0
}

func newEventQueue() *eventQueue {
	evque := &eventQueue{
		evList: queue.New(),
		coList: queue.New(),
	}

	evque.cond = sync.NewCond(&evque.guard)
	return evque
}

type eventScheduler struct {
	coPool       *queue.Queue
	queue        *eventQueue
	curCoroutine *Coroutine
	coCount      int
	maxCoCount   int
	maxCoId      int
	working      bool
	closed       bool
	mtx          sync.Mutex

	eventHandlers map[int][]*handler
	eventMap      map[uint64]*handler
	maxHid        uint64
	exitSignal    chan struct{}
}

func (es *eventScheduler) String() string {
	return fmt.Sprintf("q=%s, working=%s, closed=%s", es.queue, es.working, es.closed)
}

func (es *eventScheduler) yield() {
	es.curCoroutine.Yield()
}

func (es *eventScheduler) resume() {
	es.curCoroutine.Resume(1)
}

func (es *eventScheduler) await(f func()) {
	curCo := es.curCoroutine

	var coEv interface{} = nil
	if es.coPool.Length() == 0 {
		es.newCo()
		coEv = es.coPool.Remove()
	} else {
		coEv = es.coPool.Remove()
	}
	co := coEv.(*Coroutine)
	es.curCoroutine = co
	co.Resume(struct{}{})
	f()
	es.queue.push(curCo)
	curCo.Yield()
}

func (es *eventScheduler) handleEvent(evId int, f func(IEvent)) uint64 {

	es.maxHid += 1
	h := &handler{id: es.maxHid, evId: evId, cb: f}

	em, ok := es.eventHandlers[evId]
	if !ok {
		em = make([]*handler, 0)
	}

	em = append(em, h)
	es.eventHandlers[evId] = em
	es.eventMap[h.id] = h

	return h.id
}

func (es *eventScheduler) delHandler(id uint64) {

	if h, ok := es.eventMap[id]; ok {
		delete(es.eventMap, id)

		if hs, ok2 := es.eventHandlers[h.evId]; ok2 {
			index := -1
			for i, h2 := range hs {
				if h2.id == h.id {
					index = i
					break
				}
			}

			if index >= 0 {
				es.eventHandlers[h.evId] = append(hs[:index], hs[index+1:]...)
			}
		}
	}

}

func (es *eventScheduler) onEvent(ev IEvent) {

	if ev == nil {
		return
	}

	evId := ev.GetEventId()

	if hs, ok := es.eventHandlers[evId]; ok {

		for _, h := range hs {
			utils.CatchPanic(func() {
				h.cb(ev)
			})
		}
	}
}

func (es *eventScheduler) postEvent(ev IEvent) {
	if ev == nil {
		return
	}
	es.queue.push(ev)
}

func (es *eventScheduler) newCo() {

	for i := 0; i < 10; i++ {
		co := newCoroutine()
		es.maxCoId += 1
		co.id = es.maxCoId
		es.coPool.Add(co)

		go func(co2 *Coroutine) {
			co2.Yield()
			es.doScheduler()
			es.coCount--

		}(co)
	}

	es.coCount += 10
	log.Printf("newCo %d", es.coCount)
}

func (es *eventScheduler) doScheduler() {
	for {
		ev := es.queue.pop()
		if ev == nil {
			log.Printf("doScheduler es.queue.pop %s", ev)
			continue
		}
		evId := ev.GetEventId()
		if evId == consts.ClOSE_EVQ_EVT {

		} else if evId != consts.COROUTINE_EVENT {
			if evId == consts.CALLATER_EVT {
				if e, ok := ev.(*callLaterEvent); ok {
					utils.CatchPanic(e.f)
				} else {
					log.Printf("evq doScheduler CALLATER_EVT not a callLaterEvent")
				}
			} else {
				es.onEvent(ev)
			}
		} else {
			if co, ok := ev.(*Coroutine); ok {
				curCo := es.curCoroutine
				es.curCoroutine = co
				es.coPool.Add(curCo)
				co.Resume(struct{}{})
				curCo.Yield()
			} else {
				log.Printf("evq doScheduler COROUTINE_EVENT not a Coroutine")
			}
		}

		if es.closed && es.queue.empty() {
			close(es.exitSignal)
			return
		}

	}
}

func (es *eventScheduler) start() {

	es.mtx.Lock()
	if es.working {
		es.mtx.Unlock()
		return
	}
	es.working = true
	es.closed = false
	es.mtx.Unlock()

	es.exitSignal = make(chan struct{})
	es.coPool = queue.New()
	es.queue = newEventQueue()

	es.newCo()
	coEv := es.coPool.Remove()
	co := coEv.(*Coroutine)
	es.curCoroutine = co
	co.Resume(struct{}{})
}

func (es *eventScheduler) stop() {
	es.mtx.Lock()
	if !es.closed {
		es.postEvent(&closeEvent{})
		es.closed = true
		es.working = false
	} else {
		es.mtx.Unlock()
		return
	}
	es.mtx.Unlock()

	es.waitClear()
}

func (es *eventScheduler) waitClear() {
	select {
	case <-es.exitSignal:
		return
	case <-time.NewTimer(10 * time.Second).C:
		log.Printf("eventScheduler waitClear timeout queue=%s", es)
		return
	}
}

func Start() {
	mainEvScheduler.start()
}

func Stop() {
	mainEvScheduler.stop()
	//mainEvScheduler.waitClear()
}

func PostEvent(ev IEvent) {
	mainEvScheduler.postEvent(ev)
}

func HandleEvent(evid int, f func(event IEvent)) {
	mainEvScheduler.handleEvent(evid, f)
}

func Await(f func()) {
	mainEvScheduler.await(f)
}

func CallLater(f func()) {
	ev := newCallLaterEvent(f)
	mainEvScheduler.postEvent(ev)
}
