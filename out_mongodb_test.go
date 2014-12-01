package main

import (
	. "github.com/smartystreets/goconvey/convey"
	mgo "gopkg.in/mgo.v2"
	"testing"
	"time"
)

func TestCreateAndInsert(t *testing.T) {
	cf := map[string]string{
		"tag":         "test",
		"host":        "localhost",
		"port":        "27017",
		"database":    "test",
		"collection":  "test",
		"capped":      "on",
		"capped_size": "1024",
	}

	Convey("Test create and insert ops", t, func() {
		mongo := new(outputMongo)
		mongo.Init(cf)
		inChan := make(chan *PipelinePack, 1)
		oRunner := NewOutputRunner(inChan)
		pack := new(PipelinePack)
		pack.Msg.Data = map[string]string{
			"data":  "test",
			"hello": "world",
		}
		go mongo.Run(oRunner)

		session, err := mgo.Dial(cf["host"] + ":" + cf["port"])
		if err != nil {
			So(err.Error(), ShouldEqual, "no reachable servers")
			return
		}
		So(session, ShouldNotEqual, nil)
		defer session.Close()
		coll := session.DB(cf["database"]).C(cf["collection"])
		coll.DropCollection()
		So(coll, ShouldNotEqual, nil)

		inChan <- pack
		time.Sleep(1 * time.Second)

		result := make(map[string]string)
		err1 := coll.Find(nil).One(&result)
		So(err1, ShouldEqual, nil)
		So(result["data"], ShouldEqual, "test")
		So(result["hello"], ShouldEqual, "world")
		coll.DropCollection()
	})
}
