package backends

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/jmhodges/levigo"
	"github.com/msiebuhr/MetricBase"
	"strconv"
	"strings"
)

func serializeMetric(m MetricBase.Metric) (key []byte, value []byte) {
	// Encode key
	key = []byte(fmt.Sprintf("%v:%013d", m.Name, m.Time))

	// Encode value
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, m.Value)
	value = buf.Bytes()

	return key, value
}

// {{{
// Serialize metrics
func parseMetricKey(b []byte) (error, string, int64) {
	parts := strings.SplitN(string(b), ":", 2)

	name := parts[0]
	time, err := strconv.ParseInt(parts[1], 10, 64)

	return err, name, time
}

func parseMetricValue(b []byte) float64 {
	var r float64
	buf := bytes.NewBuffer(b)
	_ = binary.Read(buf, binary.LittleEndian, &r)
	return r
}

// }}}

type LevelDb struct {
	store *levigo.DB

	addRequests  chan MetricBase.Metric
	listRequests chan MetricBase.ListRequest
	dataRequests chan MetricBase.DataRequest

	stopChan chan bool
}

func CreateLevelDb(filename string) *LevelDb {
	options := levigo.NewOptions()
	options.SetCreateIfMissing(true)

	db, err := levigo.Open(filename, options)

	if err != nil {
		panic(err)
	}

	ls := &LevelDb{
		addRequests:  make(chan MetricBase.Metric, 100),
		listRequests: make(chan MetricBase.ListRequest, 10),
		dataRequests: make(chan MetricBase.DataRequest, 10),
		stopChan:     make(chan bool),
		store:        db,
	}

	return ls
}

func (ls *LevelDb) Start() {
	// Start listener-loop
	go func() {
		for {
			select {
			case <-ls.stopChan:
				ls.store.Close()
				return
			case metric := <-ls.addRequests:
				ls.addMetric(metric)
			case query := <-ls.listRequests:
				ls.listMetrics(query)
			case query := <-ls.dataRequests:
				ls.handleData(query)
			}
		}
	}()
}

func (s *LevelDb) Add(req MetricBase.AddRequest) {
	go func() {
		for m := range req.Data {
			s.addRequests <- m
		}
	}()
}

func (s *LevelDb) List(req MetricBase.ListRequest) {
	s.listRequests <- req
}

func (s *LevelDb) GetMetric(req MetricBase.DataRequest) {
	s.dataRequests <- req
}

func (s *LevelDb) Stop() {
	s.stopChan <- true
}

func (s *LevelDb) addMetric(metric MetricBase.Metric) {
	wo := levigo.NewWriteOptions()
	defer wo.Close()
	k, v := serializeMetric(metric)
	_ = s.store.Put(wo, k, v)
}

func (s *LevelDb) listMetrics(query MetricBase.ListRequest) {
	ro := levigo.NewReadOptions()
	ro.SetFillCache(false)
	iter := s.store.NewIterator(ro)
	defer iter.Close()

	iter.Seek([]byte{0x00})

	var currentName string = ""

	for iter = iter; iter.Valid(); iter.Next() {
		err, name, _ := parseMetricKey(iter.Key())

		// Ignore errors
		if err != nil {
			continue
		}

		// Ignore similar names.
		if name != currentName {
			query.Result <- name
			currentName = name
		}
	}

	close(query.Result)
}

func (s *LevelDb) Data(query MetricBase.DataRequest) {
	s.dataRequests <- query
}

func (s *LevelDb) handleData(query MetricBase.DataRequest) {
	ro := levigo.NewReadOptions()
	ro.SetFillCache(false)
	iter := s.store.NewIterator(ro)
	defer iter.Close()

	iter.Seek([]byte(fmt.Sprintf("%v:", query.Name)))
	for iter = iter; iter.Valid(); iter.Next() {
		err, name, time := parseMetricKey(iter.Key())
		value := parseMetricValue(iter.Value())

		if name != query.Name {
			break
		}

		if err != nil {
			continue
		}

		query.Result <- MetricBase.MetricValues{Time: time, Value: value}
	}
	close(query.Result)
}

// NO-OP
func (s *LevelDb) SetBackend(b MetricBase.Backend) {}