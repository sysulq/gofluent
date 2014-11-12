package plugins

import (
	"testing"
	"io"
	"io/ioutil"
	"strings"
	fileid "github.com/moriyoshi/go-fileid"
)

func Test_MyBufferedReader(t *testing.T) {
	inner := strings.NewReader("aaaaaaaaaaaa\nbbbb\nc")
	target := NewMyBufferedReader(inner, 16, 0)
	line, ispfx, tryAgain, err := target.ReadLine()
	if err != nil { t.FailNow() }
	if ispfx { t.Log(14); t.Fail() }
	if tryAgain { t.Log(15); t.Fail() }
	if target.position != 13 { t.Log(16); t.Fail() }
	t.Log(string(line))
	if string(line) != "aaaaaaaaaaaa" { t.Log(18); t.Fail() }

	line, ispfx, tryAgain, err = target.ReadLine()
	if err != nil { t.FailNow() }
	if ispfx { t.Log(22); t.Fail() }
	if tryAgain { t.Log(23); t.Fail() }
	if target.position != 18 { t.Log(24); t.Fail() }
	t.Log(string(line))
	if string(line) != "bbbb" { t.Log(26); t.Fail() }

	line, ispfx, tryAgain, err = target.ReadLine()
	if err != nil { t.FailNow() }
	if ispfx { t.Log(30); t.Fail() }
	if !tryAgain { t.Log(31); t.Fail() }
	if target.position != 18 { t.Log(32); t.Fail() }
	if len(line) != 0 { t.Log(33); t.Fail() }

	line, ispfx, tryAgain, err = target.ReadLine()
	if err != nil { t.FailNow() }
	if ispfx { t.Log(30); t.Fail() }
	if !tryAgain { t.Log(31); t.Fail() }
	if target.position != 18 { t.Log(32); t.Fail() }
	if len(line) != 0 { t.Log(33); t.Fail() }
}

func Test_MyBufferedReader_EOF(t *testing.T) {
	inner := strings.NewReader("aaaaaaaaaaaa\nbbbb\nc\n")
	target := NewMyBufferedReader(inner, 16, 0)
	line, ispfx, tryAgain, err := target.ReadLine()
	if err != nil { t.FailNow() }
	if ispfx { t.Log(14); t.Fail() }
	if tryAgain { t.Log(15); t.Fail() }
	if target.position != 13 { t.Log(16); t.Fail() }
	t.Log(string(line))
	if string(line) != "aaaaaaaaaaaa" { t.Log(18); t.Fail() }

	line, ispfx, tryAgain, err = target.ReadLine()
	if err != nil { t.FailNow() }
	if ispfx { t.Log(22); t.Fail() }
	if tryAgain { t.Log(23); t.Fail() }
	if target.position != 18 { t.Log(24); t.Fail() }
	t.Log(string(line))
	if string(line) != "bbbb" { t.Log(26); t.Fail() }

	line, ispfx, tryAgain, err = target.ReadLine()
	if err != nil { t.FailNow() }
	if ispfx { t.Log(30); t.Fail() }
	if tryAgain { t.Log(31); t.Fail() }
	if target.position != 20 { t.Log(32); t.Fail() }
	t.Log(string(line))
	if string(line) != "c" { t.Log(33); t.Fail() }

	line, ispfx, tryAgain, err = target.ReadLine()
	if err != io.EOF { t.FailNow() }
	if ispfx { t.Log(30); t.Fail() }
	if tryAgain { t.Log(31); t.Fail() }
	if target.position != 20 { t.Log(32); t.Fail() }
	t.Log(string(line))
	if string(line) != "" { t.Log(33); t.Fail() }
}


func Test_MyBufferedReader_crlf(t *testing.T) {
	inner := strings.NewReader("aaaaaaaaaaaa\r\nbbbb\r\nc")
	target := NewMyBufferedReader(inner, 16, 0)
	line, ispfx, tryAgain, err := target.ReadLine()
	if err != nil { t.FailNow() }
	if ispfx { t.Log(48); t.Fail() }
	if tryAgain { t.Log(49); t.Fail() }
	if target.position != 14 { t.Log(50); t.Fail() }
	t.Log(string(line))
	if string(line) != "aaaaaaaaaaaa" { t.Log(52); t.Fail() }

	line, ispfx, tryAgain, err = target.ReadLine()
	if err != nil { t.FailNow() }
	if ispfx { t.Log(56); t.Fail() }
	if tryAgain { t.Log(57); t.Fail() }
	if target.position != 20 { t.Log(58); t.Fail() }
	t.Log(string(line))
	if string(line) != "bbbb" { t.Log(60); t.Fail() }

	line, ispfx, tryAgain, err = target.ReadLine()
	if err != nil { t.FailNow() }
	if ispfx { t.Log(64); t.Fail() }
	if !tryAgain { t.Log(65); t.Fail() }
	if target.position != 20 { t.Log(66); t.Fail() }
	if len(line) != 0 { t.Log(67); t.Fail() }

	line, ispfx, tryAgain, err = target.ReadLine()
	if err != nil { t.FailNow() }
	if ispfx { t.Log(64); t.Fail() }
	if !tryAgain { t.Log(65); t.Fail() }
	if target.position != 20 { t.Log(66); t.Fail() }
	if len(line) != 0 { t.Log(67); t.Fail() }
}

func Test_MyBufferedReader_pfx(t *testing.T) {
	inner := strings.NewReader("aaaaaaaa\nbbbb")
	target := NewMyBufferedReader(inner, 8, 0)
	line, ispfx, tryAgain, err := target.ReadLine()
	if err != nil { t.FailNow() }
	if !ispfx { t.Log(82); t.Fail() }
	if tryAgain { t.Log(83); t.Fail() }
	if target.position != 8 { t.Log(84); t.Fail() }
	t.Log(string(line))
	if string(line) != "aaaaaaaa" { t.Log(86); t.Fail() }

	line, ispfx, tryAgain, err = target.ReadLine()
	if err != nil { t.FailNow() }
	if ispfx { t.Log(90); t.Fail() }
	if tryAgain { t.Log(91); t.Fail() }
	if target.position != 9 { t.Log(92); t.Fail() }
	t.Log(string(line))
	if string(line) != "" { t.Log(94); t.Fail() }

	line, ispfx, tryAgain, err = target.ReadLine()
	if err != nil { t.FailNow() }
	if ispfx { t.Log(98); t.Fail() }
	if !tryAgain { t.Log(99); t.Fail() }
	if target.position != 9 { t.Log(100); t.Fail() }
	if len(line) != 0 { t.Log(101); t.Fail() }

	line, ispfx, tryAgain, err = target.ReadLine()
	if err != nil { t.FailNow() }
	if ispfx { t.Log(98); t.Fail() }
	if !tryAgain { t.Log(99); t.Fail() }
	if target.position != 9 { t.Log(100); t.Fail() }
	if len(line) != 0 { t.Log(101); t.Fail() }
}

func Test_marshalledSizeOfPositionFileData(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "in_tail")
	if err != nil {
		t.FailNow()
	}
	defer tempFile.Close()
	id , err := fileid.GetFileId(tempFile.Name(), true)
	if err != nil {
		t.FailNow()
	}
	blob := marshalPositionFileData(&tailPositionFileData {
		Path: tempFile.Name(),
		Position: 1000,
		Id: id,
	})
	t.Log(string(blob))
	x := tailPositionFileData {}
	consumed, err := unmarshalPositionFileData(&x, blob)
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}
	t.Logf("%d %d", consumed, len(blob))
	if consumed != len(blob) {
		t.Fail()
	}
	if x.Position != 1000 { t.Fail() }
	if !fileid.IsSame(x.Id, id) { t.Fail() }
}

func Test_buildTagFromPath(t *testing.T) {
	var result string
	result = buildTagFromPath("a/b/c")
	t.Log(result)
	if result != "a.b.c" {
		t.Fail()
	}
	result = buildTagFromPath("a//b/c")
	t.Log(result)
	if result != "a.b.c" {
		t.Fail()
	}
	result = buildTagFromPath("/a//b/c")
	t.Log(result)
	if result != "a.b.c" {
		t.Fail()
	}
}
