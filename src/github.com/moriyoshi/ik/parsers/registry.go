package parsers

import "github.com/moriyoshi/ik"

var _plugins []ik.LineParserPlugin = make([]ik.LineParserPlugin, 0)

func AddPlugin(plugin ik.LineParserPlugin) bool {
	_plugins = append(_plugins, plugin)
	return false
}

func GetPlugins() []ik.LineParserPlugin {
	return _plugins
}
