// @APIVersion 1.0.0
// @Title KYC Service
// @Description Gnosis KYC Service for PM
// @Contact denis@gnosis.pm
package routers

import (
	"github.com/astaxie/beego"
	"github.com/gnosis/pm-kyc-service/controllers"
)

func init() {
	beego.SetStaticPath("/swagger", "swagger")

	ns :=
		beego.NewNamespace("/v1",
			beego.NSInclude(
				&controllers.UserController{},
			),
		)

	beego.AddNamespace(ns)

}
