package ik

import (
	"net/http"
	"strings"
	"testing"
	"bytes"
	"os"
)

type myOpener string
type myFile struct { b *bytes.Buffer }
type myFileSystem struct { s string }

func (f *myFile) Close() error { return nil }
func (f *myFile) Read(b []byte) (int, error) { return f.b.Read(b) }
func (f *myFile) Readdir(int) ([]os.FileInfo, error) { return nil, nil }
func (f *myFile) Seek(int64, int) (int64, error) { return 0, nil }
func (f *myFile) Stat() (os.FileInfo, error) { return nil, nil }

func (fs *myFileSystem) Open(string) (http.File, error) {
	return &myFile{ b: bytes.NewBuffer(([]byte)(fs.s)) }, nil
}

func (opener myOpener) NewLineReader(filename string) (LineReader, error) {
	return NewDefaultLineReader(filename, strings.NewReader(string(opener))), nil
}

func (s myOpener) FileSystem() http.FileSystem  { return &myFileSystem { s: string(s) } }
func (myOpener) BasePath() string             { return "" }
func (myOpener) NewOpener(path string) Opener { return myOpener("") }

func TestParseConfig(t *testing.T) {
	const data = "<test>\n" +
		"attr_name1 attr_value1\n" +
		"attr_name2 attr_value2\n" +
		"</test>\n"
	config, err := ParseConfig(myOpener(data), "test.cfg")
	if err != nil {
		panic(err.Error())
	}
	if len(config.Root.Elems) != 1 {
		t.Fail()
	}
	if config.Root.Elems[0].Name != "test" {
		t.Fail()
	}
	if len(config.Root.Elems[0].Attrs) != 2 {
		t.Fail()
	}
	if config.Root.Elems[0].Attrs["attr_name1"] != "attr_value1" {
		t.Fail()
	}
	if config.Root.Elems[0].Attrs["attr_name2"] != "attr_value2" {
		t.Fail()
	}
}

// vim: sts=4 sw=4 ts=4 noet
