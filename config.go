package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
)

type Opener interface {
	FileSystem() http.FileSystem
	BasePath() string
	NewOpener(path string) Opener
}

type Config struct {
	Root *ConfigElement
}

type ConfigElement struct {
	Name  string
	Args  string
	Attrs map[string]string
	Elems []*ConfigElement
}

type LineReader interface {
	Next() (string, error)
	Close() error
	Filename() string
	LineNumber() int
}

type parserContext struct {
	tag     string
	tagArgs string
	elems   []*ConfigElement
	attrs   map[string]string
	opener  Opener
}

type DefaultLineReader struct {
	filename   string
	lineNumber int
	inner      *bufio.Reader
	closer     io.Closer
}

type DefaultOpener http.Dir

var (
	stripCommentRegexp = regexp.MustCompile("\\s*(?:#.*)?$")
	startTagRegexp     = regexp.MustCompile("^<([a-zA-Z0-9_]+)\\s*(.+?)?>$")
	attrRegExp         = regexp.MustCompile("^([a-zA-Z0-9_]+)\\s+(.*)$")
)

func (reader *DefaultLineReader) Next() (string, error) {
	reader.lineNumber += 1
	line, isPrefix, err := reader.inner.ReadLine()
	if isPrefix {
		return "", errors.New(fmt.Sprintf("Line too long in %s at line %d", reader.filename, reader.lineNumber))
	}
	if err != nil {
		return "", err
	}
	return string(line), err
}

func (reader *DefaultLineReader) Close() error {
	if reader.closer != nil {
		return reader.closer.Close()
	}
	return nil
}

func (reader *DefaultLineReader) Filename() string {
	return reader.filename
}

func (reader *DefaultLineReader) LineNumber() int {
	return reader.lineNumber
}

func NewDefaultLineReader(filename string, reader io.Reader) *DefaultLineReader {
	closer, ok := reader.(io.Closer)
	if !ok {
		closer = nil
	}
	return &DefaultLineReader{
		filename:   filename,
		lineNumber: 0,
		inner:      bufio.NewReader(reader),
		closer:     closer,
	}
}

func (opener DefaultOpener) FileSystem() http.FileSystem {
	return http.Dir(opener)
}

func (opener DefaultOpener) BasePath() string {
	return string(opener)
}

func (opener DefaultOpener) NewOpener(path_ string) Opener {
	if !path.IsAbs(path_) {
		path_ = path.Join(string(opener), path_)
	}
	return DefaultOpener(path_)
}

func NewLineReader(opener Opener, filename string) (LineReader, error) {
	//file, err := opener.FileSystem().Open(filename)
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	return NewDefaultLineReader(filename, file), nil
}

func (opener DefaultOpener) Open(filename string) (http.File, error) {
	return http.Dir(opener).Open(filename)
}

func makeParserContext(tag string, tagArgs string, opener Opener) *parserContext {
	return &parserContext{
		tag:     tag,
		tagArgs: tagArgs,
		elems:   make([]*ConfigElement, 0),
		attrs:   make(map[string]string),
		opener:  opener,
	}
}

func makeConfigElementFromContext(context *parserContext) *ConfigElement {
	return &ConfigElement{
		Name:  context.tag,
		Args:  context.tagArgs,
		Attrs: context.attrs,
		Elems: context.elems,
	}
}

func handleInclude(reader LineReader, context *parserContext, attrValue string) error {
	url_, err := url.Parse(attrValue)
	if err != nil {
		return err
	}
	if url_.Scheme == "file" || url_.Path == attrValue {
		var abspathPattern string
		if path.IsAbs(url_.Path) {
			abspathPattern = url_.Path
		} else {
			abspathPattern = path.Join(context.opener.BasePath(), url_.Path)
		}
		files, err := Glob(context.opener.FileSystem(), abspathPattern)
		if err != nil {
			return err
		}
		for _, file := range files {
			newReader, err := NewLineReader(context.opener, file)
			if err != nil {
				return err
			}
			defer newReader.Close()
			parseConfig(newReader, &parserContext{
				tag:     context.tag,
				tagArgs: context.tagArgs,
				elems:   context.elems,
				attrs:   context.attrs,
				opener:  context.opener.NewOpener(path.Dir(file)),
			})
		}
		return nil
	} else {
		return errors.New("Not implemented!")
	}
}

func handleSpecialAttrs(reader LineReader, context *parserContext, attrName string, attrValue string) (bool, error) {
	if attrName == "include" {
		return false, handleInclude(reader, context, attrValue)
	}
	return false, nil
}

func parseConfig(reader LineReader, context *parserContext) error {
	tagEnd := "</" + context.tag + ">"
	for {
		line, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}

		line = strings.TrimLeft(line, " \t\r\n")
		line = stripCommentRegexp.ReplaceAllLiteralString(line, "")
		if len(line) == 0 {
			continue
		} else if submatch := startTagRegexp.FindStringSubmatch(line); submatch != nil {
			subcontext := makeParserContext(
				submatch[1],
				submatch[2],
				nil,
			)
			err = parseConfig(reader, subcontext)
			if err != nil {
				return err
			}
			context.elems = append(context.elems, makeConfigElementFromContext(subcontext))
		} else if line == tagEnd {
			break
		} else if submatch := attrRegExp.FindStringSubmatch(line); submatch != nil {
			handled, err := handleSpecialAttrs(reader, context, submatch[1], submatch[2])
			if err != nil {
				return err
			}
			if !handled {
				context.attrs[submatch[1]] = submatch[2]
			}
		} else {
			return errors.New(fmt.Sprintf("Parse error in %s at line %s", reader.Filename(), reader.LineNumber()))
		}
	}
	return nil
}

func ParseConfig(opener Opener, filename string) (*Config, error) {
	context := makeParserContext("(root)", "", opener)
	reader, err := NewLineReader(nil, filename)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	err = parseConfig(reader, context)
	if err != nil {
		return nil, err
	}
	return &Config{Root: makeConfigElementFromContext(context)}, nil
}

func DefaultGC() *GlobalConfig {
	gc := new(GlobalConfig)
	gc.PoolSize = 100
	return gc
}
