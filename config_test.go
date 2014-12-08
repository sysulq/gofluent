package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"os"
	"testing"
)

func Test_ReadConf(t *testing.T) {
	content := `<source>
  type tail
  path /var/log/apache2.log
  format /^(?<host>[^ ]*)$/
  pos_file /var/log/apache2.pos
  tag test
</source>

<match test.**>
  type httpsqs
  host localhost
  auth 123456
  flush_interval 10
</match>`

	testFile := "/tmp/test.file"
	os.Remove(testFile)

	Convey("Write config to a file and read from it", t, func() {
		Convey("Write config to a file", func() {
			f, err := os.OpenFile(testFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
			So(err, ShouldEqual, nil)
			_, err = f.WriteString(content)
			So(err, ShouldEqual, nil)
			f.Close()
		})

		Convey("Read from it", func() {
			config, err := ParseConfig(nil, testFile)
			So(err, ShouldEqual, nil)
			So(config, ShouldNotEqual, nil)
			for _, v := range config.Root.Elems {
				if v.Name == "source" {
					So(v.Attrs["type"], ShouldEqual, "tail")
					So(v.Attrs["path"], ShouldEqual, "/var/log/apache2.log")
					So(v.Attrs["format"], ShouldEqual, "/^(?<host>[^ ]*)$/")
					So(v.Attrs["pos_file"], ShouldEqual, "/var/log/apache2.pos")
					So(v.Attrs["tag"], ShouldEqual, "test")
				} else if v.Name == "match" {
					So(v.Args, ShouldEqual, "test.**")
					So(v.Attrs["type"], ShouldEqual, "httpsqs")
					So(v.Attrs["host"], ShouldEqual, "localhost")
					So(v.Attrs["auth"], ShouldEqual, "123456")
					So(v.Attrs["flush_interval"], ShouldEqual, "10")
				}
			}
		})
	})
	os.Remove(testFile)
}
