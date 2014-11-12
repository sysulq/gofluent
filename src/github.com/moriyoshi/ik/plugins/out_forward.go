package plugins

import (
	"github.com/moriyoshi/ik"
	"bytes"
	"github.com/ugorji/go/codec"
	"log"
	"net"
	"reflect"
	"strconv"
	"time"
)

type ForwardOutput struct {
	factory *ForwardOutputFactory
	logger  *log.Logger
	codec   *codec.MsgpackHandle
	bind    string
	enc     *codec.Encoder
	conn    net.Conn
	buffer  bytes.Buffer
}

func (output *ForwardOutput) encodeEntry(tag string, record ik.TinyFluentRecord) error {
	v := []interface{} { tag, record.Timestamp, record.Data }
	if output.enc == nil {
		output.enc = codec.NewEncoder(&output.buffer, output.codec)
	}
	err := output.enc.Encode(v)
	if err != nil {
		return err
	}
	return err
}

func (output *ForwardOutput) encodeRecordSet(recordSet ik.FluentRecordSet) error {
	v := []interface{} { recordSet.Tag, recordSet.Records }
	if output.enc == nil {
		output.enc = codec.NewEncoder(&output.buffer, output.codec)
	}
	err := output.enc.Encode(v)
	if err != nil {
		return err
	}
	return err
}

func (output *ForwardOutput) flush() error {
	if output.conn == nil {
		conn, err := net.Dial("tcp", output.bind)
		if err != nil {
			output.logger.Printf("%#v", err.Error())
			return err
		} else {
			output.conn = conn
		}
	}
	n, err := output.buffer.WriteTo(output.conn)
	if err != nil {
		output.logger.Printf("Write failed. size: %d, buf size: %d, error: %#v", n, output.buffer.Len(), err.Error())
		output.conn = nil
		return err
	}
	if n > 0 {
		output.logger.Printf("Forwarded: %d bytes (left: %d bytes)\n", n, output.buffer.Len())
	}
	output.conn.Close()
	output.conn = nil
	return nil
}

func (output *ForwardOutput) run_flush(flush_interval int) {
	ticker := time.NewTicker(time.Duration(flush_interval) * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				output.flush()
			}
		}
	}()
}

func (output *ForwardOutput) Emit(recordSet []ik.FluentRecordSet) error {
	for _, recordSet := range recordSet {
		err := output.encodeRecordSet(recordSet)
		if err != nil {
			output.logger.Printf("%#v", err)
			return err
		}
	}
	return nil
}

func (output *ForwardOutput) Factory() ik.Plugin {
	return output.factory
}

func (output *ForwardOutput) Run() error {
	time.Sleep(1000000000)
	return ik.Continue
}

func (output *ForwardOutput) Shutdown() error {
	return nil
}

type ForwardOutputFactory struct {
}

func newForwardOutput(factory *ForwardOutputFactory, logger *log.Logger, bind string) (*ForwardOutput, error) {
	_codec := codec.MsgpackHandle{}
	_codec.MapType = reflect.TypeOf(map[string]interface{}(nil))
	_codec.RawToString = false
	_codec.StructToArray = true
	return &ForwardOutput{
		factory: factory,
		logger:  logger,
		codec:   &_codec,
		bind:    bind,
	}, nil
}

func (factory *ForwardOutputFactory) Name() string {
	return "forward"
}

func (factory *ForwardOutputFactory) New(engine ik.Engine, config *ik.ConfigElement) (ik.Output, error) {
	host, ok := config.Attrs["host"]
	if !ok {
		host = "localhost"
	}
	netPort, ok := config.Attrs["port"]
	if !ok {
		netPort = "24224"
	}
	flush_interval_str, ok := config.Attrs["flush_interval"]
	if !ok {
		flush_interval_str = "60"
	}
	flush_interval, err := strconv.Atoi(flush_interval_str)
	if err != nil {
		engine.Logger().Print(err.Error())
		return nil, err
	}
	bind := host + ":" + netPort
	output, err := newForwardOutput(factory, engine.Logger(), bind)
	output.run_flush(flush_interval)
	return output, err
}

func (factory *ForwardOutputFactory) BindScorekeeper(scorekeeper *ik.Scorekeeper) {
}

var _ = AddPlugin(&ForwardOutputFactory{})
