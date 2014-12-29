package main

import (
	mgo "gopkg.in/mgo.v2"
	"log"
	"strconv"
)

type outputMongo struct {
	host         string
	port         string
	database     string
	collection   string
	user         string
	password     string
	capped       bool
	capped_size  int
	failed_count int
}

func (this *outputMongo) Init(cf map[string]string) error {
	this.host = "localhost"
	this.port = "27017"
	this.capped = false
	this.failed_count = 0

	value := cf["host"]
	if len(value) > 0 {
		this.host = value
	}

	value = cf["port"]
	if len(value) > 0 {
		this.port = value
	}

	value = cf["database"]
	if len(value) > 0 {
		this.database = value
	}

	value = cf["collection"]
	if len(value) > 0 {
		this.collection = value
	}

	value = cf["user"]
	if len(value) > 0 {
		this.user = value
	}

	value = cf["password"]
	if len(value) > 0 {
		this.password = value
	}

	value = cf["capped"]
	if len(value) > 0 {
		if value == "on" {
			this.capped = true
		}
	}

	value = cf["capped_size"]
	if len(value) > 0 {
		this.capped_size, _ = strconv.Atoi(value)
	}

	return nil
}

func (this *outputMongo) Run(runner OutputRunner) error {

	//[mongodb://][user:pass@]host1[:port1][,host2[:port2],...][/database][?options]
	url := "mongodb://"
	if len(this.user) != 0 && len(this.password) != 0 {
		url += this.user + ":" + this.password + "@"
	}
	url += this.host + ":" + this.port + "/" + this.database

	session, err := mgo.Dial(url)
	if err != nil {
		log.Println("mgo.Dial failed, err:", err)
		return err
	}

	info := &mgo.CollectionInfo{
		Capped:   this.capped,
		MaxBytes: this.capped_size * 1024 * 1024,
	}

	coll := session.DB(this.database).C(this.collection)
	err = coll.Create(info)
	if err != nil {
		return err
	}

	for {
		select {
		case pack := <-runner.InChan():
			{

				session.Refresh()
				coll := session.DB(this.database).C(this.collection)

				err = coll.Insert(pack.Msg.Data)
				if err != nil {
					this.failed_count++
					log.Println("insert failed, count=", this.failed_count, "err:", err)
					pack.Recycle()
					continue
				}

				pack.Recycle()
			}
		}
	}
}

func init() {
	RegisterOutput("mongodb", func() interface{} {
		return new(outputMongo)
	})
}
