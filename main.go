package main

import (
	"github.com/astaxie/beego"
	_ "github.com/gnosis/pm-kyc-service/routers"
)

func main() {
	beego.Run()
}
