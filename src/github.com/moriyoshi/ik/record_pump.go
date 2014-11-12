package ik

import (
	"time"
)

type RecordPump struct {
	port      Port
	ch        chan FluentRecord
	control   chan bool
	buffer    map[string]*FluentRecordSet
	heartbeat *time.Ticker
}

func (pump *RecordPump) Port() Port {
	return pump.port
}

func (pump *RecordPump) EmitOne(record FluentRecord) {
	pump.ch <- record
}

func (pump *RecordPump) flush() error {
	recordSets := make([]FluentRecordSet, 0, len(pump.buffer))
	for _, recordSet := range pump.buffer {
		recordSets = append(recordSets, *recordSet)
	}
	pump.buffer = make(map[string]*FluentRecordSet)
	return pump.port.Emit(recordSets)
}

func (pump *RecordPump) Run() error {
	for {
		select {
		case record := <-pump.ch:
			buffer := pump.buffer
			recordSet, ok := buffer[record.Tag]
			if !ok {
				recordSet = &FluentRecordSet {
					Tag: record.Tag,
					Records: make([]TinyFluentRecord, 0, 16),
				}
				buffer[record.Tag] = recordSet
			}
			recordSet.Records = append(recordSet.Records, TinyFluentRecord {
				Timestamp: record.Timestamp,
				Data: record.Data,
			})
			break
		case <-pump.heartbeat.C:
			err := pump.flush()
			if err != nil {
				return err
			}
			break
		case needsToBeStopped := <-pump.control:
			if needsToBeStopped {
				pump.heartbeat.Stop()
			}
			return pump.flush()
		}
	}
	return Continue

}

func (pump *RecordPump) Shutdown() error {
	pump.control <- true
	return nil
}

func NewRecordPump(port Port, backlog int) *RecordPump {
	return &RecordPump {
		port:      port,
		ch:        make(chan FluentRecord, backlog),
		control:   make(chan bool, 1),
		buffer:    make(map[string]*FluentRecordSet),
		heartbeat: time.NewTicker(time.Duration(1000000000)),
	}
}
