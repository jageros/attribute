package main

import (
	"github.com/jageros/attribute"
	"log"
)

type dbConfig struct{}

func (df *dbConfig) GetType() string {
	return "mongodb"
}

func (df *dbConfig) GetAddr() string {
	return "127.0.0.1:27017"
}
func (df *dbConfig) GetDB() string {
	return "test_db1"
}
func (df *dbConfig) GetUser() string {
	return ""
}

func (df *dbConfig) GetPassword() string {
	return ""
}

func main() {
	attribute.Start(&dbConfig{})
	attr := attribute.NewAttrMgr("myTest", 10086)
	attr.SetInt("val", 10009)
	err := attr.Save(true)
	if err != nil {
		log.Printf("Save Err=%v", err)
	}
	attribute.Stop()
}
