package plugins

import "github.com/moriyoshi/ik"

var _plugins []ik.Plugin = make([]ik.Plugin, 0)

func AddPlugin(plugin ik.Plugin) bool {
	_plugins = append(_plugins, plugin)
	return false
}

func GetPlugins() []ik.Plugin {
	return _plugins
}
