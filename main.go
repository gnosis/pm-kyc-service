package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/plugins/cors"
	_ "github.com/gnosis/pm-kyc-service/models"
	_ "github.com/gnosis/pm-kyc-service/routers"
)

func main() {
	log := logs.NewLogger(10000)
	log.SetLogger("console")

	onlyCompile, _ := strconv.ParseBool(os.Getenv("ONLY_COMPILE"))

	fmt.Println("Configure Cors")
	beego.InsertFilter("*", beego.BeforeRouter, cors.Allow(&cors.Options{
		// AllowOrigins:     []string{"*"},
		AllowAllOrigins:	true,
		AllowMethods:     []string{"POST", "PUT", "PATCH"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))
	fmt.Println("Start running")
	if onlyCompile {
		beego.BConfig.RunMode = beego.DEV
		go beego.Run()
		time.Sleep(5000 * time.Millisecond)
		log.Warn("Exiting program")
		os.Exit(0)
	} else {
		beego.Run()
	}
}
