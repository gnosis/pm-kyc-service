package main

import (
	"github.com/astaxie/beego"
	_ "github.com/gnosis/pm-kyc-service/routers"
	"github.com/astaxie/beego/logs"
)

func main() {
	log := logs.NewLogger(10000)
	log.SetLogger("console")
	beego.Run()
}
