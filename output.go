package main

import (
	"fmt"
	"os"
	"regexp"
)

type output interface {
	new() interface{}
	start(ctx chan Context) error
	configure(f map[string]string) error
}

var output_plugins = make(map[string]output)

func RegisterOutput(name string, out output) {
	if out == nil {
		Log("output: Register output is nil")
	}

	if _, ok := output_plugins[name]; ok {
		Log("output: Register called twice for output " + name)
	}

	output_plugins[name] = out
}

func NewOutputs(ctx chan Context) error {
	outChan := make([]chan Context, 0)
	for _, output_config := range config.Outputs_config {
		f := output_config.(map[string]string)
		tmpch := make(chan Context)
		outChan = append(outChan, tmpch)
		go func(f map[string]string, tmpch chan Context) {
			output_type, ok := f["type"]
			if !ok {
				fmt.Println("no type configured")
				os.Exit(-1)
			}

			output_plugin, ok := output_plugins[output_type]
			if !ok {
				Log("unkown type ", output_type)
				os.Exit(-1)
			}

			out := output_plugin.new()
			err := out.(output).configure(f)
			if err != nil {
				Log(err)
			}

			err = out.(output).start(tmpch)
			if err != nil {
				Log(err)
			}
		}(f, tmpch)
	}

	for {
		s := <-ctx
		for i, o := range outChan {
			f := config.Outputs_config[i].(map[string]string)
			tag := f["tag"]

			chunk, err := BuildRegexpFromGlobPattern(tag)
			if err != nil {
				return err
			}

			re, err := regexp.Compile(chunk)
			if err != nil {
				return err
			}

			flag := re.MatchString(s.tag)
			if flag == false {
				continue
			}

			o <- s
		}
	}

	return nil
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
