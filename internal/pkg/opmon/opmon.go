package opmon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/jageros/attribute/internal/pkg/utils"
	"log"
	"net/http"
	"sync"
	"time"
)

var (
	operationAllocPool = sync.Pool{
		New: func() interface{} {
			return &Operation{}
		},
	}

	monitor        = newMonitor()
	falconAgentUrl string
	dumpInterval   int
)

func Initialize(appName string, appID uint32, dumpInterval_ int, falconAgentPort int) {
	dumpInterval = dumpInterval_
	if dumpInterval > 0 {
		if falconAgentPort > 0 {
			falconAgentUrl = fmt.Sprintf("http://127.0.0.1:%d/v1/push", falconAgentPort)
		}
		monitor.appName = appName
		monitor.appID = appID
		monitor.endpoint = fmt.Sprintf("%s%d", appName, appID)
		go func() {
			for {
				time.Sleep(time.Duration(dumpInterval) * time.Second)
				utils.CatchPanic(monitor.Dump)
			}
		}()
	}
}

type _OpInfo struct {
	count         uint64
	totalDuration time.Duration
	maxDuration   time.Duration
}

type _Monitor struct {
	sync.Mutex
	appName  string
	appID    uint32
	endpoint string
	opInfos  map[string]*_OpInfo
}

func newMonitor() *_Monitor {
	m := &_Monitor{
		opInfos: map[string]*_OpInfo{},
	}
	return m
}

func (monitor *_Monitor) record(opname string, duration time.Duration) {
	monitor.Lock()
	info := monitor.opInfos[opname]
	if info == nil {
		info = &_OpInfo{}
		monitor.opInfos[opname] = info
	}
	info.count += 1
	info.totalDuration += duration
	if duration > info.maxDuration {
		info.maxDuration = duration
	}
	monitor.Unlock()
}

func (monitor *_Monitor) Dump() {
	type _T struct {
		name string
		info *_OpInfo
	}
	var opInfos map[string]*_OpInfo
	monitor.Lock()
	opInfos = monitor.opInfos
	monitor.opInfos = map[string]*_OpInfo{} // clear to be empty
	monitor.Unlock()

	if len(opInfos) <= 0 {
		return
	}

	var copyOpInfos []_T
	for name, opinfo := range opInfos {
		copyOpInfos = append(copyOpInfos, _T{name, opinfo})
	}
	//sort.Slice(copyOpInfos, func(i, j int) bool {
	//	_t1 := copyOpInfos[i]
	//	_t2 := copyOpInfos[j]
	//	return _t1.name < _t2.name
	//})
	var falconInfos []map[string]interface{}
	ts := time.Now().Unix()
	for _, _t := range copyOpInfos {
		opname, opinfo := _t.name, _t.info
		if falconAgentUrl != "" {
			falconInfos = append(falconInfos, map[string]interface{}{
				"endpoint":  monitor.endpoint,
				"metric":    opname + "_" + "count",
				"timestamp": ts,
				"step":      dumpInterval,
				"value":     opinfo.count,
			})

			falconInfos = append(falconInfos, map[string]interface{}{
				"endpoint":  monitor.endpoint,
				"metric":    opname + "_" + "avgMillisecond",
				"timestamp": ts,
				"step":      dumpInterval,
				"value":     (opinfo.totalDuration / time.Duration(opinfo.count)).Nanoseconds() / 1000000,
			})

			falconInfos = append(falconInfos, map[string]interface{}{
				"endpoint":  monitor.endpoint,
				"metric":    opname + "_" + "maxMillisecond",
				"timestamp": ts,
				"step":      dumpInterval,
				"value":     opinfo.maxDuration.Nanoseconds() / 1000000,
			})

		} else {
			log.Printf("monitor Dump ==== %-30sx%-10d AVG %-10s MAX %-10s", opname, opinfo.count,
				opinfo.totalDuration/time.Duration(opinfo.count), opinfo.maxDuration)
		}
	}

	if len(falconInfos) > 0 {
		payload, _ := json.Marshal(falconInfos)
		_, err := http.Post(falconAgentUrl, "", bytes.NewReader(payload))
		if err != nil {
			log.Printf("Post falconAgent %s fail %s", falconAgentUrl, err)
			return
		}
	}
}

type Operation struct {
	name      string
	startTime time.Time
}

func StartOperation(operationName string) *Operation {
	op := operationAllocPool.Get().(*Operation)
	op.name = operationName
	op.startTime = time.Now()
	return op
}

func (op *Operation) Finish(warnThreshold time.Duration) {
	takeTime := time.Now().Sub(op.startTime)
	monitor.record(op.name, takeTime)
	if warnThreshold > 0 && takeTime >= warnThreshold {
		log.Printf("opmon: operation %s takes %s > %s", op.name, takeTime, warnThreshold)
	}
	operationAllocPool.Put(op)
}
