package main

import (
	"net/http"
	"path"
	"regexp"
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

type PatternError struct {
	message string
}

func (self *PatternError) Error() string {
	return self.message
}

func buildRegexpFromGlobPatternInner(pattern string, startPos int) (string, int, error) {
	state := 0
	chunk := ""

	patternLength := len(pattern)
	var i int
	for i = startPos; i < patternLength; i += 1 {
		if state == 0 {
			c := pattern[i]
			if c == '*' {
				state = 2
			} else if c == '}' || c == ',' {
				break
			} else if c == '{' {
				chunk += "(?:"
				first := true
				for {
					i += 1
					subchunk, lastPos, err := buildRegexpFromGlobPatternInner(pattern, i)
					if err != nil {
						return "", 0, err
					}
					if lastPos == patternLength {
						return "", 0, &PatternError{"unexpected end of pattern (in expectation of '}' or ',')"}
					}
					if !first {
						chunk += "|"
					}
					i = lastPos
					chunk += "(?:" + subchunk + ")"
					first = false
					if pattern[lastPos] == '}' {
						break
					} else if pattern[lastPos] != ',' {
						return "", 0, &PatternError{"never get here"}
					}
				}
				chunk += ")"
			} else {
				chunk += regexp.QuoteMeta(string(c))
			}
		} else if state == 1 {
			// escape
			c := pattern[i]
			chunk += regexp.QuoteMeta(string(c))
			state = 0
		} else if state == 2 {
			c := pattern[i]
			if c == '*' {
				state = 3
			} else {
				chunk += "[^.]*" + regexp.QuoteMeta(string(c))
				state = 0
			}
		} else if state == 3 {
			// recursive any
			c := pattern[i]
			if c == '*' {
				return "", 0, &PatternError{"unexpected *"}
			} else if c == '.' {
				chunk += "(?:.*\\.|^)"
			} else {
				chunk += ".*" + regexp.QuoteMeta(string(c))
			}
			state = 0
		}
	}
	if state == 2 {
		chunk += "[^.]*"
	} else if state == 3 {
		chunk += ".*"
	}
	return chunk, i, nil
}

func BuildRegexpFromGlobPattern(pattern string) (string, error) {
	chunk, pos, err := buildRegexpFromGlobPatternInner(pattern, 0)
	if err != nil {
		return "", err
	}
	if pos != len(pattern) {
		return "", &PatternError{"unexpected '" + string(pattern[pos]) + "'"}
	}
	return "^" + chunk + "$", nil
}
