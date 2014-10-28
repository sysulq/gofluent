package main

import (
	"net/http"
	"path"
	"strings"
)

type globMatcherContext struct {
	fs               http.FileSystem
	path             string
	restOfComponents []string
	resultCollector  func(string) error
}

func doMatch(context globMatcherContext) error {
	if len(context.restOfComponents) == 0 {
		return context.resultCollector(context.path)
	}

	f, err := context.fs.Open(context.path)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}

	entries, err := f.Readdir(-1)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		name := entry.Name()
		matched, err := path.Match(context.restOfComponents[0], name)
		if err != nil {
			return err
		}
		if matched {
			err := doMatch(globMatcherContext{
				fs:               context.fs,
				path:             path.Join(context.path, name),
				restOfComponents: context.restOfComponents[1:],
				resultCollector:  context.resultCollector,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func Glob(fs http.FileSystem, pattern string) ([]string, error) {
	retval := make([]string, 0)
	components := strings.Split(pattern, "/")
	var path_ string
	if len(components) > 0 && components[0] == "" {
		path_ = "/"
		components = components[1:]
	} else {
		path_ = "."
	}
	return retval, doMatch(globMatcherContext{
		fs:               fs,
		path:             path_,
		restOfComponents: components,
		resultCollector: func(match string) error {
			retval = append(retval, match)
			return nil
		},
	})
}
