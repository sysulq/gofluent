package ik

import (
	"log"
	"time"
	"math/rand"
	"github.com/moriyoshi/ik/task"
)

type recurringTaskDaemon struct {
	engine *engineImpl
	shutdown bool
}

func (daemon *recurringTaskDaemon) Run() error {
	daemon.engine.recurringTaskScheduler.ProcessEvent()
	if daemon.shutdown {
		return nil
	}
	time.Sleep(1000000000)
	return Continue
}


func (daemon *recurringTaskDaemon) Shutdown() error {
	daemon.shutdown = true
	daemon.engine.recurringTaskScheduler.NoOp()
	return nil
}

type engineImpl struct {
	logger          *log.Logger
	opener          Opener
	lineParserPluginRegistry LineParserPluginRegistry
	randSource	rand.Source
	scorekeeper     *Scorekeeper
	defaultPort     Port
	spawner         *Spawner
	pluginInstances []PluginInstance
	taskRunner      task.TaskRunner
	recurringTaskScheduler *task.RecurringTaskScheduler
}

func (engine *engineImpl) Logger() *log.Logger {
	return engine.logger
}

func (engine *engineImpl) Opener() Opener {
	return engine.opener
}

func (engine *engineImpl) LineParserPluginRegistry() LineParserPluginRegistry {
	return engine.lineParserPluginRegistry
}

func (engine *engineImpl) RandSource() rand.Source {
	return engine.randSource
}

func (engine *engineImpl) Scorekeeper() *Scorekeeper {
	return engine.scorekeeper
}

func (engine *engineImpl) SpawneeStatuses() ([]SpawneeStatus, error) {
	return engine.spawner.GetSpawneeStatuses()
}

func (engine *engineImpl) DefaultPort() Port {
	return engine.defaultPort
}

func (engine *engineImpl) Dispose() error {
	spawnees, err := engine.spawner.GetRunningSpawnees()
	if err != nil {
		return err
	}
	for _, spawnee := range spawnees {
		engine.spawner.Kill(spawnee)
	}
	engine.spawner.PollMultiple(spawnees)
	return nil
}

func (engine *engineImpl) Spawn(spawnee Spawnee) error {
	return engine.spawner.Spawn(spawnee)
}

func (engine *engineImpl) Launch(pluginInstance PluginInstance) error {
	var err error
	spawnee, ok := pluginInstance.(Spawnee)
	if ok {
		err = engine.Spawn(spawnee)
		if err != nil {
			return err
		}
	}
	engine.pluginInstances = append(engine.pluginInstances, pluginInstance)
	return nil
}

func (engine *engineImpl) PluginInstances() []PluginInstance {
	retval := make([]PluginInstance, len(engine.pluginInstances))
	copy(retval, engine.pluginInstances)
	return retval
}

func (engine *engineImpl) RecurringTaskScheduler() *task.RecurringTaskScheduler {
	return engine.recurringTaskScheduler
}

func (engine *engineImpl) Start() error {
	spawnees, err := engine.spawner.GetRunningSpawnees()
	if err != nil {
		return err
	}
	return engine.spawner.PollMultiple(spawnees)
}

func NewEngine(logger *log.Logger, opener Opener, lineParserPluginRegistry LineParserPluginRegistry, scorekeeper *Scorekeeper, defaultPort Port) *engineImpl {
	taskRunner := &task.SimpleTaskRunner {}
	recurringTaskScheduler := task.NewRecurringTaskScheduler(
		func () time.Time { return time.Now() },
		taskRunner,
	)
	engine := &engineImpl{
		logger:          logger,
		opener:          opener,
		lineParserPluginRegistry: lineParserPluginRegistry,
		randSource:      NewRandSourceWithTimestampSeed(),
		scorekeeper:     scorekeeper,
		defaultPort:     defaultPort,
		spawner:         NewSpawner(),
		pluginInstances: make([]PluginInstance, 0),
		taskRunner:      taskRunner,
		recurringTaskScheduler: recurringTaskScheduler,
	}
	engine.Spawn(&recurringTaskDaemon { engine, false })
	return engine
}
