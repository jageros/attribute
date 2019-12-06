package main

import (
	"github.com/jager/attribute"
	"github.com/jager/attribute/example/conf"
	"time"
)

func main() {
	attribute.Start(conf.GetDBCfg())
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
	attr.Save(false)
	//attr.Load()
	//log.Printf("attr=%+v", attr)

	attribute.Stop()
}
