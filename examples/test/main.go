package main

import (
	"fmt"
	"github.com/jageros/attribute"
	"github.com/jageros/db"
	"github.com/jageros/group"
	"log"
)

func main() {
	g := group.Default()
	attribute.Initialize(g, db.MongoDBConfig("127.0.0.1:27017", "test_db1"))
	attr := attribute.NewAttrMgr("myTest", 10086)
	attr.SetInt("val", 10009)
	err := attr.Save(true)
	if err != nil {
		log.Printf("Save Err=%v", err)
	}
	err = g.Wait()
	fmt.Println(err)
}
