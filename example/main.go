package main

import (
	"github.com/jageros/attribute"
	"github.com/jageros/attribute/example/conf"
	"log"
	"time"
)

func main() {
	attribute.ConfigMongoDB(conf.GetDBCfg())
	attr := attribute.NewAttrMgr("actor", 1)
	attr.SetStr("name", "jager")
	mAttr := attr.GetMapAttr("msg")
	if mAttr == nil {
		mAttr = attribute.NewMapAttr()
		attr.SetMapAttr("msg", mAttr)
	}
	mAttr.SetInt("age", 26)
	mAttr.SetInt64("time", time.Now().Unix())
	mAttr.SetStr("addr", "广东广州")
	mAttr.SetStr("email", "lhj168os@gmail.com")
	mAttr.SetStr("github", "https://github.com/jageros")
	mAttr.SetStr("blog", "https://blog.csdn.net/lhj_168")
	err := attr.Save(true)
	if err != nil {
		log.Printf("MongoDB save error: %v", err)
	}else {
		log.Printf("MongoDB save successful !")
	}
	//attr.Load()
	//log.Printf("attr=%+v", attr)

	attribute.Stop()
}
