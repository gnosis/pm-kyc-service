package models

import (
	"os"
	"strconv"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
	_ "github.com/lib/pq"           // Postgres for production
	_ "github.com/mattn/go-sqlite3" // Sqlite3 for fast tests
)

// Check represents
type OnfidoCheck struct {
	CheckId    string      `orm:"pk;size(50)"`
	IsVerified bool        `orm:"default(false)"`
	IsClear    bool        `orm:"default(false)"`
	User       *OnfidoUser `orm:"null;rel(one);"`
}

// User represents the Prediction Markets user that must follow a KYC process in order to use the official frontend
type OnfidoUser struct {
	EthereumAddress string       `orm:"pk;size(40)"`
	ApplicantId     string       `orm:"size(50);unique"`
	TermsHash       string       `orm:"size(64)"`
	TermsSignature  string       `orm:"size(130);unique"`
	OnfidoCheck     *OnfidoCheck `orm:"reverse(one)"`
}

func init() {
	orm.RegisterModel(new(OnfidoUser), new(OnfidoCheck))
	migrateDatabase, _ := strconv.ParseBool(beego.AppConfig.String("migrateDatabase"))
	if migrateDatabase {
		// set default database
		orm.RegisterDataBase("default",
			beego.AppConfig.String("database"),
			beego.AppConfig.String("databaseParams"),
		)

		// Error.
		err := orm.RunSyncdb("default", false, true) // database, force, verbose
		if err != nil {
			logs.Error(err.Error())
			os.Exit(1)
		}
	}
}
