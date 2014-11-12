package ik

import (
	"regexp"
)

type fluentRouterRule struct {
	re   *regexp.Regexp
	port Port
}

type FluentRouter struct {
	rules []*fluentRouterRule
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

func (router *FluentRouter) AddRule(pattern string, port Port) error {
	chunk, err := BuildRegexpFromGlobPattern(pattern)
	if err != nil {
		return err
	}
	re, err := regexp.Compile(chunk)
	if err != nil {
		return err
	}
	newRule := &fluentRouterRule{re, port}
	router.rules = append(router.rules, newRule)
	return nil
}

func (router *FluentRouter) Emit(recordSets []FluentRecordSet) error {
	recordSetsMap := make(map[Port][]FluentRecordSet)
	for i := range recordSets {
		recordSet := &recordSets[i]
		for _, rule := range router.rules {
			if rule.re.MatchString(recordSet.Tag) {
				recordSetsForPort, ok := recordSetsMap[rule.port]
				if !ok {
					recordSetsForPort = make([]FluentRecordSet, 0)
				} else {
					lastRecordSets := &recordSetsForPort[len(recordSetsForPort) - 1]
					if &lastRecordSets.Records == &recordSet.Records {
						continue
					}
				}
				recordSetsMap[rule.port] = append(recordSetsForPort, *recordSet)
			}
		}
	}
	for port, recordSets := range recordSetsMap {
		err := port.Emit(recordSets)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewFluentRouter() *FluentRouter {
	return &FluentRouter{make([]*fluentRouterRule, 0)}
}
