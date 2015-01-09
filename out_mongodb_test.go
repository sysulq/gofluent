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
	mongo := new(outputMongo)
	mongo.Init(cf)
	pack := new(PipelinePack)
	pack.Msg.Data = map[string]interface{}{
		"data":  "test",
		"hello": "world",
	}
	inChan := make(chan *PipelinePack, 1)
	oRunner := NewOutputRunner(inChan)
	inChan <- pack

	go mongo.Run(oRunner)
	time.Sleep(1 * time.Second)

	Convey("Test create and insert ops", t, func() {

		//[mongodb://][user:pass@]host1[:port1][,host2[:port2],...][/database][?options]
		url := "mongodb://" +
			cf["host"] + ":" + cf["port"] + "/" + cf["database"]

		session, err := mgo.Dial(url)
		if err != nil {
			So(err.Error(), ShouldEqual, "no reachable servers")
			return
		}
		So(session, ShouldNotEqual, nil)
		defer session.Close()
		coll := session.DB(cf["database"]).C(cf["collection"])
		So(coll, ShouldNotEqual, nil)

		result := make(map[string]string)
		err1 := coll.Find(nil).One(&result)
		So(err1, ShouldEqual, nil)
		So(result["data"], ShouldEqual, "test")
		So(result["hello"], ShouldEqual, "world")
		coll.DropCollection()
	})
}
