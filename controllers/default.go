package controllers

import (
	"github.com/astaxie/beego"
)

// Operations about users
type UserController struct {
	beego.Controller
}

type json_struct struct {
	Hello string `json:"hello"`
}

// @Title Get User
// @Description Retrieves user
// @Success 200
// @Failure 403 body is empty
// @router /users/:address [get]
func (controller *UserController) Get() {
	m := json_struct{controller.Ctx.Input.Param(":address")}
	controller.Data["json"] = &m
	controller.ServeJSON()
}
