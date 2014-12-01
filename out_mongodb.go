package main

import (
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"strconv"
	"time"
)

type outputMongo struct {
	host           string
	port           string
	database       string
	collection     string
	user           string
	password       string
	capped         bool
	capped_size    int
	flush_interval int
	count          int
	buffer         map[string][]bson.M
}

func (this *outputMongo) Init(cf map[string]string) error {
	this.host = "localhost"
	this.port = "27017"
	this.capped = false
	this.flush_interval = 1
	this.count = 0
	this.buffer = make(map[string][]bson.M, 0)

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

	value = cf["flush_interval"]
	if len(value) > 0 {
		this.flush_interval, _ = strconv.Atoi(value)
	}

	return nil
}

func (this *outputMongo) Run(runner OutputRunner) error {
	tick := time.NewTicker(time.Second * time.Duration(this.flush_interval))

	session, err := mgo.Dial(this.host + ":" + this.port)
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
	session.Close()

	for {
		select {

		case pack := <-runner.InChan():
			{
				doc := make(bson.M, 0)
				for k, v := range pack.Msg.Data {
					doc[k] = v
				}
				this.buffer[pack.Msg.Tag] = append(this.buffer[pack.Msg.Tag], doc)
				this.count++
				pack.Recycle()
			}
		case <-tick.C:
			{

				if len(this.buffer) > 0 {
					this.flush()
				}
			}
		}
	}
}

func (this *outputMongo) flush() error {
	session, err := mgo.Dial(this.host + ":" + this.port)
	if err != nil {
		log.Println("mgo.Dial failed, err:", err)
		return err
	}
	defer session.Close()

	coll := session.DB(this.database).C(this.collection)
	for k, v := range this.buffer {
		log.Println("[mongodb] k:", k)

		for _, m := range v {
			err = coll.Insert(m)
			if err != nil {
				log.Println("insert failed, err:", err)
				break
			}
		}
		if err != nil {
			return err
		}
		log.Println("[mongodb] insert success, count=", this.count)

		this.buffer[k] = this.buffer[k][0:0]
		delete(this.buffer, k)
		this.count = 0
	}
	return err
}

func init() {
	RegisterOutput("mongodb", func() interface{} {
		return new(outputMongo)
	})
}
