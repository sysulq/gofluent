package plugins

import (
	"github.com/moriyoshi/ik"
	"fmt"
	"log"
	"os"
	"time"
)

type StdoutOutput struct {
	factory *StdoutOutputFactory
	logger  *log.Logger
}

func (output *StdoutOutput) Emit(recordSets []ik.FluentRecordSet) error {
	for _, recordSet := range recordSets {
		for _, record := range recordSet.Records {
			fmt.Fprintf(os.Stdout, "%d %s: %s\n", record.Timestamp, recordSet.Tag, record.Data)
		}
	}
	return nil
}

func (output *StdoutOutput) Factory() ik.Plugin {
	return output.factory
}

func (output *StdoutOutput) Run() error {
	time.Sleep(1000000000)
	return ik.Continue
}

func (output *StdoutOutput) Shutdown() error {
	return nil
}

func (output *StdoutOutput) Dispose() {
	output.Shutdown()
}

type StdoutOutputFactory struct {
}

func newStdoutOutput(factory *StdoutOutputFactory, logger *log.Logger) (*StdoutOutput, error) {
	return &StdoutOutput{
		factory: factory,
		logger:  logger,
	}, nil
}

func (factory *StdoutOutputFactory) Name() string {
	return "stdout"
}

func (factory *StdoutOutputFactory) New(engine ik.Engine, _ *ik.ConfigElement) (ik.Output, error) {
	return newStdoutOutput(factory, engine.Logger())
}

func (factory *StdoutOutputFactory) BindScorekeeper(scorekeeper *ik.Scorekeeper) {
}

var _ = AddPlugin(&StdoutOutputFactory{})
