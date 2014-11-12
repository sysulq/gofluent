package plugins

import (
	"github.com/moriyoshi/ik"
	"errors"
	"fmt"
	"github.com/ugorji/go/codec"
	"io"
	"log"
	"net"
	"reflect"
	"strconv"
	"sync/atomic"
)

type forwardClient struct {
	input  *ForwardInput
	logger *log.Logger
	conn   net.Conn
	codec  *codec.MsgpackHandle
	enc    *codec.Encoder
	dec    *codec.Decoder
}

type ForwardInput struct {
	factory  *ForwardInputFactory
	port     ik.Port
	logger   *log.Logger
	bind     string
	listener net.Listener
	codec    *codec.MsgpackHandle
	clients  map[net.Conn]*forwardClient
	entries  int64
}

type EntryCountTopic struct {}

type ConnectionCountTopic struct {}

type ForwardInputFactory struct {
}

func coerceInPlace(data map[string]interface{}) {
	for k, v := range data {
		switch v_ := v.(type) {
		case []byte:
			data[k] = string(v_) // XXX: byte => rune
		case map[string]interface{}:
			coerceInPlace(v_)
		}
	}
}

func decodeRecordSet(tag []byte, entries []interface{}) (ik.FluentRecordSet, error) {
	records := make([]ik.TinyFluentRecord, len(entries))
	for i, _entry := range entries {
		entry, ok := _entry.([]interface{})
		if !ok {
			return ik.FluentRecordSet {}, errors.New("Failed to decode recordSet")
		}
		timestamp, ok := entry[0].(uint64)
		if !ok {
			return ik.FluentRecordSet {}, errors.New("Failed to decode timestamp field")
		}
		data, ok := entry[1].(map[string]interface{})
		if !ok {
			return ik.FluentRecordSet {}, errors.New("Failed to decode data field")
		}
		coerceInPlace(data)
		records[i] = ik.TinyFluentRecord {
			Timestamp: timestamp,
			Data:      data,
		}
	}
	return ik.FluentRecordSet {
		Tag:       string(tag), // XXX: byte => rune
		Records:   records,
	}, nil
}

func (c *forwardClient) decodeEntries() ([]ik.FluentRecordSet, error) {
	v := []interface{}{nil, nil, nil}
	err := c.dec.Decode(&v)
	if err != nil {
		return nil, err
	}
	tag, ok := v[0].([]byte)
	if !ok {
		return nil, errors.New("Failed to decode tag field")
	}

	var retval []ik.FluentRecordSet
	switch timestamp_or_entries := v[1].(type) {
	case uint64:
		timestamp := timestamp_or_entries
		data, ok := v[2].(map[string]interface{})
		if !ok {
			return nil, errors.New("Failed to decode data field")
		}
		coerceInPlace(data)
		retval = []ik.FluentRecordSet {
			{
				Tag:       string(tag), // XXX: byte => rune
				Records: []ik.TinyFluentRecord {
					{
						Timestamp: timestamp,
						Data:      data,
					},
				},
			},
		}
	case float64:
		timestamp := uint64(timestamp_or_entries)
		data, ok := v[2].(map[string]interface{})
		if !ok {
			return nil, errors.New("Failed to decode data field")
		}
		retval = []ik.FluentRecordSet {
			{
				Tag:       string(tag), // XXX: byte => rune
				Records: []ik.TinyFluentRecord {
					{
						Timestamp: timestamp,
						Data:      data,
					},
				},
			},
		}
	case []interface{}:
		if !ok {
			return nil, errors.New("Unexpected payload format")
		}
		recordSet, err := decodeRecordSet(tag, timestamp_or_entries)
		if err != nil {
			return nil, err
		}
		retval = []ik.FluentRecordSet { recordSet }
	case []byte:
		entries := make([]interface{}, 0)
		err := codec.NewDecoderBytes(timestamp_or_entries, c.codec).Decode(&entries)
		if err != nil {
			return nil, err
		}
		recordSet, err := decodeRecordSet(tag, entries)
		if err != nil {
			return nil, err
		}
		retval = []ik.FluentRecordSet { recordSet }
	default:
		return nil, errors.New(fmt.Sprintf("Unknown type: %t", timestamp_or_entries))
	}
	atomic.AddInt64(&c.input.entries, int64(len(retval)))
	return retval, nil
}


func handleInner(c *forwardClient) bool {
	recordSets, err := c.decodeEntries()
	defer func() {
		if len(recordSets) > 0 {
			err_ := c.input.Port().Emit(recordSets)
			if err_ != nil {
				c.logger.Print(err_.Error())
			}
		}
	}()
	if err == nil {
		return true;
	}

	err_, ok := err.(net.Error)
	if ok {
		if err_.Temporary() {
			c.logger.Println("Temporary failure: %s", err_.Error())
			return true
		}
	}
	if err == io.EOF {
		c.logger.Printf("Client %s closed the connection", c.conn.RemoteAddr().String())
	} else {
		c.logger.Print(err.Error())
	}
	return false
}

func (c *forwardClient) handle() {
	for handleInner(c) {}
	err := c.conn.Close()
	if err != nil {
		c.logger.Print(err.Error())
	}
	c.input.markDischarged(c)
}

func newForwardClient(input *ForwardInput, logger *log.Logger, conn net.Conn, _codec *codec.MsgpackHandle) *forwardClient {
	c := &forwardClient{
		input:  input,
		logger: logger,
		conn:   conn,
		codec:  _codec,
		enc:    codec.NewEncoder(conn, _codec),
		dec:    codec.NewDecoder(conn, _codec),
	}
	input.markCharged(c)
	return c
}

func (input *ForwardInput) Factory() ik.Plugin {
	return input.factory
}

func (input *ForwardInput) Port() ik.Port {
	return input.port
}

func (input *ForwardInput) Run() error {
	conn, err := input.listener.Accept()
	if err != nil {
		input.logger.Print(err.Error())
		return err
	}
	go newForwardClient(input, input.logger, conn, input.codec).handle()
	return ik.Continue
}

func (input *ForwardInput) Shutdown() error {
	for conn, _ := range input.clients {
		err := conn.Close()
		if err != nil {
			input.logger.Printf("Error during closing connection: %s", err.Error())
		}
	}
	return input.listener.Close()
}

func (input *ForwardInput) Dispose() {
	input.Shutdown()
}

func (input *ForwardInput) markCharged(c *forwardClient) {
	input.clients[c.conn] = c
}

func (input *ForwardInput) markDischarged(c *forwardClient) {
	delete(input.clients, c.conn)
}

func newForwardInput(factory *ForwardInputFactory, logger *log.Logger, engine ik.Engine, bind string, port ik.Port) (*ForwardInput, error) {
	_codec := codec.MsgpackHandle{}
	_codec.MapType = reflect.TypeOf(map[string]interface{}(nil))
	_codec.RawToString = false
	listener, err := net.Listen("tcp", bind)
	if err != nil {
		logger.Print(err.Error())
		return nil, err
	}
	return &ForwardInput{
		factory:  factory,
		port:     port,
		logger:   logger,
		bind:     bind,
		listener: listener,
		codec:    &_codec,
		clients:  make(map[net.Conn]*forwardClient),
		entries:  0,
	}, nil
}

func (factory *ForwardInputFactory) Name() string {
	return "forward"
}

func (factory *ForwardInputFactory) New(engine ik.Engine, config *ik.ConfigElement) (ik.Input, error) {
	listen, ok := config.Attrs["listen"]
	if !ok {
		listen = ""
	}
	netPort, ok := config.Attrs["port"]
	if !ok {
		netPort = "24224"
	}
	bind := listen + ":" + netPort
	return newForwardInput(factory, engine.Logger(), engine, bind, engine.DefaultPort())
}

func (factory *ForwardInputFactory) BindScorekeeper(scorekeeper *ik.Scorekeeper) {
	scorekeeper.AddTopic(ik.ScorekeeperTopic {
		Plugin: factory,
		Name: "entries",
		DisplayName: "Total number of entries",
		Description: "Total number of entries received so far",
		Fetcher: &EntryCountTopic {},
	})
	scorekeeper.AddTopic(ik.ScorekeeperTopic {
		Plugin: factory,
		Name: "connections",
		DisplayName: "Connections",
		Description: "Number of connections currently handled",
		Fetcher: &ConnectionCountTopic {},
	})
}

func (topic *EntryCountTopic) Markup(input_ ik.PluginInstance) (ik.Markup, error) {
	text, err := topic.PlainText(input_)
	if err != nil {
		return ik.Markup {}, err
	}
	return ik.Markup { []ik.MarkupChunk { { Text: text } } }, nil
}

func (topic *EntryCountTopic) PlainText(input_ ik.PluginInstance) (string, error) {
	input := input_.(*ForwardInput)
	return strconv.FormatInt(input.entries, 10), nil
}

func (topic *ConnectionCountTopic) Markup(input_ ik.PluginInstance) (ik.Markup, error) {
	text, err := topic.PlainText(input_)
	if err != nil {
		return ik.Markup {}, err
	}
	return ik.Markup { []ik.MarkupChunk { { Text: text } } }, nil
}

func (topic *ConnectionCountTopic) PlainText(input_ ik.PluginInstance) (string, error) {
	input := input_.(*ForwardInput)
	return strconv.Itoa(len(input.clients)), nil // XXX: race
}

var _ = AddPlugin(&ForwardInputFactory{})
