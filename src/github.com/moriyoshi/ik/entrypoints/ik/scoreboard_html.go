package main

import (
	"log"
	"time"
	"strconv"
	"net"
	"net/http"
	"html/template"
	"unsafe"
	"bytes"
	"reflect"
	"sync/atomic"
	"fmt"
	"html"
	"github.com/moriyoshi/ik"
	"github.com/moriyoshi/ik/markup"
)

var mainTemplate = `<!DOCTYPE html>
<html>
<head>
<title>Ik Scoreboard</title>
<style type="text/css">
body {
  margin: 0 0;
  background-color: #eee;
  color: #222;
}

header {
  background-color: #888;
  padding: 4px 8px;
}

main {
  padding: 8px 8px;
}

h1 {
  margin: 0 0;
}

.table {
  margin: 0 0;
  padding: 0 0;
  border: 1px solid #ccc;
  border-collapse: collapse;
}

.table.fullwidth {
  width: 100%;
}

.table > thead > tr > td,
.table > thead > tr > th {
  background-color: #888;
  color: #eee;
}

.table > thead > tr > th,
.table > tbody > tr > td,
.table > tbody > tr > th {
  border: 1px solid #ccc;
  padding: 4px 4px;
}

.exitStatus.running {
  background-color: #eec;
}

.exitStatus.error {
  background-color: #ecc;
  color: #f00;
}

.pluginName.input:before,
.pluginName.output:before,
.pluginName.scoreboard:before {
	display: inline-block;
	border-radius: 4px;
	padding: 2px 4px;
	margin: 0 4px 0 0;
	font-size: 0.75em;
	font-weight: bold;
	color: #eee;
}

.pluginName.input:before {
	content: "input";
	background-color: #083;
}

.pluginName.output:before {
	content: "output";
	background-color: #d30;
}

.pluginName.scoreboard:before {
	content: "scoreboard";
	background-color: #da4;
}

</style>
</head>
<body>
<header>
<h1>Ik Scoreboard</h1>
</header>
<main>
<h2>Plugins</h2>
<dl>
<dt>Input Plugins</dt>
<dd>
{{range .InputPlugins}}
{{.Name}}</li>
{{end}}
<dt>Output Plugins</dt>
<dd>
{{range .OutputPlugins}}
{{.Name}}
{{end}}
</dd>
<dt>Scoreboard Plugins</dt>
<dd>
{{range .ScoreboardPlugins}}
{{.Name}}
{{end}}
</dd>
</dl>
<h2>Plugin Statuses</h2>
{{range $plugin, $pluginInstanceStatuses := .PluginInstanceStatusesPerPlugin}}
<h3>{{renderPluginName $plugin}}</h3>
{{range $_, $pluginInstanceStatus := $pluginInstanceStatuses}}
<h4>Instance #{{$pluginInstanceStatus.Id}}</h4>
{{with .SpawneeStatus}}
<table class="table">
  <tbody>
    <tr>
      <th>Spawnee ID</th>
      <td>{{.Id}}</td>
    </tr>
    <tr>
      <th>Status</th>
      <td class="exitStatus {{renderExitStatusStyle .ExitStatus}}">{{renderExitStatusLabel .ExitStatus}}</td>
    </tr>
  </tbody>
</table>
{{end}}
<h5>Topics</h5>
{{if len $pluginInstanceStatus.Topics}}
<table class="table">
  <thead>
    <tr>
      <th>Name</th>
      <th>Value</th>
      <th>Description</th>
    </tr>
  </thead>
  <tbody>
    {{range $pluginInstanceStatus.Topics}}
    <tr>
      <th>{{.DisplayName}} ({{.Name}})</th>
      <td>{{renderMarkup .Value}}</td>
      <td>{{.Description}}</td>
    </tr>
    {{end}}
  </tbody>
</table>
{{else}}
No topics available
{{end}}
{{end}}

{{end}}
</main>
</body>
</html>`

type HTMLHTTPScoreboard struct {
	template *template.Template
	factory  *HTMLHTTPScoreboardFactory
	logger   *log.Logger
	engine   ik.Engine
	registry ik.PluginRegistry
	listener net.Listener
	server   http.Server
	requests int64
}

type HTMLHTTPScoreboardFactory struct {
}

type pluginInstanceStatusTopic struct {
	Name string
	DisplayName string
	Description string
	Value ik.Markup
}

type pluginInstanceStatus struct {
	Id int
	PluginInstance ik.PluginInstance
	SpawneeStatus *ik.SpawneeStatus
	Topics []pluginInstanceStatusTopic
}

type viewModel struct {
	InputPlugins []ik.InputFactory
	OutputPlugins []ik.OutputFactory
	ScoreboardPlugins []ik.ScoreboardFactory
	Plugins []ik.Plugin
	PluginInstanceStatusesPerPlugin map[ik.Plugin][]pluginInstanceStatus
	SpawneeStatuses map[ik.Spawnee]ik.SpawneeStatus
}

type requestCountFetcher struct {}

func (fetcher *requestCountFetcher) Markup(scoreboard_ ik.PluginInstance) (ik.Markup, error) {
	text, err := fetcher.PlainText(scoreboard_)
	if err != nil {
		return ik.Markup {}, err
	}
	return ik.Markup { []ik.MarkupChunk { { Attrs: 0, Text: text } } }, nil
}

func (fetcher *requestCountFetcher) PlainText(scoreboard_ ik.PluginInstance) (string, error) {
	scoreboard := scoreboard_.(*HTMLHTTPScoreboard)
	return strconv.FormatInt(scoreboard.requests, 10), nil
}

func spawneeName(spawnee ik.Spawnee) string {
	switch spawnee_ := spawnee.(type) {
	case ik.PluginInstance:
		return spawnee_.Factory().Name()
	default:
		return reflect.TypeOf(spawnee_).Name()
	}
}

func renderExitStatusStyle(err error) string {
	switch err_ := err.(type) {
	case *ik.ContinueType:
		_ = err_
		return "running"
	default:
		return "error"
	}
}

func renderExitStatusLabel(err error) string {
	switch err_ := err.(type) {
	case *ik.ContinueType:
		_ = err_
		return `Running`
	default:
		return err.Error()
	}
}

func renderMarkup(markup_ ik.Markup) template.HTML {
	buf := &bytes.Buffer {}
	renderer := &markup.HTMLRenderer { Out: buf }
	renderer.Render(&markup_)
	return template.HTML(buf.String())
}

func renderPluginType(plugin ik.Plugin) string {
	switch plugin.(type) {
	case ik.InputFactory:
		return "input"
	case ik.OutputFactory:
		return "output"
	case ik.ScoreboardFactory:
		return "scoreboard"
	default:
		return "unknown"
	}
}

func renderPluginName(plugin ik.Plugin) template.HTML {
	return template.HTML(
		fmt.Sprintf(`<span class="pluginName %s">%s</span>`,
			html.EscapeString(renderPluginType(plugin)),
			html.EscapeString(html.EscapeString(plugin.Name())),
		),
	)
}

func (scoreboard *HTMLHTTPScoreboard) Run() error {
	scoreboard.server.Serve(scoreboard.listener)
	return nil
}

func (scoreboard *HTMLHTTPScoreboard) Shutdown() error {
	return scoreboard.listener.Close()
}

func (scoreboard *HTMLHTTPScoreboard) Factory() ik.Plugin {
	return scoreboard.factory
}

func (factory *HTMLHTTPScoreboardFactory) Name() string {
	return "html_http"
}

func (scoreboard *HTMLHTTPScoreboard) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	atomic.AddInt64(&scoreboard.requests, 1)
	resp.Header().Set("Content-Type", "text/html; charset=utf-8")
	resp.WriteHeader(200)
	spawneeStatuses_, err := scoreboard.engine.SpawneeStatuses()
	spawneeStatuses := make(map[ik.Spawnee]ik.SpawneeStatus)
	if err == nil {
		for _, spawneeStatus := range spawneeStatuses_ {
			spawneeStatuses[spawneeStatus.Spawnee] = spawneeStatus
		}
	}
	plugins := scoreboard.registry.Plugins()
	inputPlugins := make([]ik.InputFactory, 0)
	outputPlugins := make([]ik.OutputFactory, 0)
	scoreboardPlugins := make([]ik.ScoreboardFactory, 0)
	for _, plugin := range plugins {
		switch plugin_ := plugin.(type) {
		case ik.InputFactory:
			inputPlugins = append(inputPlugins, plugin_)
		case ik.OutputFactory:
			outputPlugins = append(outputPlugins, plugin_)
		case ik.ScoreboardFactory:
			scoreboardPlugins = append(scoreboardPlugins, plugin_)
		}
	}
	pluginInstances := scoreboard.engine.PluginInstances()
	pluginInstanceStatusesPerPlugin := make(map[ik.Plugin][]pluginInstanceStatus)
	for i, pluginInstance := range pluginInstances {
		plugin := pluginInstance.Factory()
		topics_ := scoreboard.engine.Scorekeeper().GetTopics(plugin)
		topics := make([]pluginInstanceStatusTopic, len(topics_))
		for i, topic_ := range topics_ {
			topics[i].Name = topic_.Name
			topics[i].DisplayName = topic_.DisplayName
			topics[i].Description = topic_.Description
			topics[i].Value, err = topic_.Fetcher.Markup(pluginInstance)
			if err != nil {
				errorMessage := err.Error()
				scoreboard.logger.Print(errorMessage)
				topics[i].Value = ik.Markup { []ik.MarkupChunk { { Attrs: ik.Embolden, Text: fmt.Sprintf("Error: %s", errorMessage) } } }
			}
		}
		pluginInstanceStatus_ := pluginInstanceStatus {
			Id: i + 1,
			PluginInstance: pluginInstance,
			Topics: topics,
		}
		spawneeStatus, ok := spawneeStatuses[pluginInstance]
		if ok {
			pluginInstanceStatus_.SpawneeStatus = &spawneeStatus
		}
		pluginInstanceStatuses_, ok := pluginInstanceStatusesPerPlugin[plugin]
		if !ok {
			pluginInstanceStatuses_ = make([]pluginInstanceStatus, 0)
		}
		pluginInstanceStatuses_ = append(pluginInstanceStatuses_, pluginInstanceStatus_)
		pluginInstanceStatusesPerPlugin[plugin] = pluginInstanceStatuses_
	}
	scoreboard.template.Execute(resp, viewModel {
		InputPlugins: inputPlugins,
		OutputPlugins: outputPlugins,
		ScoreboardPlugins: scoreboardPlugins,
		Plugins: plugins,
		PluginInstanceStatusesPerPlugin: pluginInstanceStatusesPerPlugin,
		SpawneeStatuses: spawneeStatuses,
	})
}

func newHTMLHTTPScoreboard(factory *HTMLHTTPScoreboardFactory, logger *log.Logger, engine ik.Engine, registry ik.PluginRegistry, bind string, readTimeout time.Duration, writeTimeout time.Duration) (*HTMLHTTPScoreboard, error) {
	template_, err := template.New("main").Funcs(template.FuncMap {
			"spawneeName": spawneeName,
			"renderExitStatusStyle": renderExitStatusStyle,
			"renderExitStatusLabel": renderExitStatusLabel,
			"renderMarkup": renderMarkup,
			"renderPluginName": renderPluginName,
			"renderPluginType": renderPluginType,
		}).Parse(mainTemplate)
	if err != nil {
		logger.Print(err.Error())
		return nil, err
	}
	server := http.Server {
		Addr:           bind,
		Handler:        nil,
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		MaxHeaderBytes: 0,
		TLSConfig:      nil,
		TLSNextProto:   nil,
	}
	listener, err := net.Listen("tcp", bind)
	if err != nil {
		logger.Print(err.Error())
		return nil, err
	}
	retval := &HTMLHTTPScoreboard {
		template: template_,
		factory:  factory,
		logger:   logger,
		engine:   engine,
		registry: registry,
		server:   server,
		listener: listener,
		requests: 0,
	}
	retval.server.Handler = retval
	return retval, nil
}

var durationSize = unsafe.Sizeof(time.Duration(0))

func (factory *HTMLHTTPScoreboardFactory) New(engine ik.Engine, registry ik.PluginRegistry, config *ik.ConfigElement) (ik.Scoreboard, error) {
	listen, ok := config.Attrs["listen"]
	if !ok {
		listen = ""
	}
	netPort, ok := config.Attrs["port"]
	if !ok {
		netPort = "24226"
	}
	bind := listen + ":" + netPort
	readTimeout := time.Duration(0)
	{
		valueStr, ok := config.Attrs["read_timeout"]
		if ok {
			value, err := strconv.ParseInt(valueStr, 10, int(durationSize * 8))
			if err != nil {
				return nil, err
			}
			readTimeout = time.Duration(value)
		}
	}
	writeTimeout := time.Duration(0)
	{
		writeTimeoutStr, ok := config.Attrs["write_timeout"]
		if ok {
			value, err := strconv.ParseInt(writeTimeoutStr, 10, int(durationSize * 8))
			if err != nil {
				return nil, err
			}
			writeTimeout = time.Duration(value)
		}
	}
	return newHTMLHTTPScoreboard(factory, engine.Logger(), engine, registry, bind, readTimeout, writeTimeout)
}

func (factory *HTMLHTTPScoreboardFactory) BindScorekeeper(scorekeeper *ik.Scorekeeper) {
	scorekeeper.AddTopic(ik.ScorekeeperTopic {
		Plugin: factory,
		Name: "requests",
		DisplayName: "Requests",
		Description: "Number of requests accepted",
		Fetcher: &requestCountFetcher {},
	})
}
