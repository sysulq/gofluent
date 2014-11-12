package ik

import (
	"log"
	"fmt"
	"errors"
)

type Scorekeeper struct {
	logger *log.Logger
	topics map[Plugin]map[string]ScorekeeperTopic
}

func (sk *Scorekeeper) GetPlugins() []Plugin {
	plugins := make([]Plugin, len(sk.topics))
	i := 0
	for plugin, _ := range sk.topics {
		plugins[i] = plugin
		i += 1
	}
	return plugins
}

func (sk *Scorekeeper) GetTopics(plugin Plugin) []ScorekeeperTopic {
	entries, ok := sk.topics[plugin]
	if !ok {
		return []ScorekeeperTopic {}
	}
	topics := make([]ScorekeeperTopic, len(entries))
	i := 0
	for _, entry := range entries {
		topics[i] = entry
		i += 1
	}
	return topics
}

func (sk *Scorekeeper) AddTopic(topic ScorekeeperTopic) {
	sk.logger.Printf("AddTopic: plugin=%s, name=%s", topic.Plugin.Name(), topic.Name)
	entries, ok := sk.topics[topic.Plugin]
	if !ok {
		entries = make(map[string]ScorekeeperTopic)
		sk.topics[topic.Plugin] = entries
	}
	entries[topic.Name] = topic
}

func (sk *Scorekeeper) Fetch(plugin Plugin, name string) (ScoreValueFetcher, error) {
	var ok bool
	var entries map[string]ScorekeeperTopic
	var entry ScorekeeperTopic
	entries, ok = sk.topics[plugin]
	if ok {
		entry, ok = entries[name]
	}
	if !ok {
		return nil, errors.New(fmt.Sprintf("unknown topic: plugin=%s, name=%s", plugin.Name(), name))
	}
	return entry.Fetcher, nil
}

func (sk *Scorekeeper) Dispose() {}

func NewScorekeeper(logger *log.Logger) *Scorekeeper {
	return &Scorekeeper{
		logger: logger,
		topics: make(map[Plugin]map[string]ScorekeeperTopic),
	}
}
