package controllers

import (
	"github.com/astaxie/beego"
	"github.com/ethereum/go-ethereum/crypto"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/validation"
	"github.com/astaxie/beego/logs"
)

// Operations about users
type UserController struct {
	beego.Controller
}

type UserPost struct {
	Email string `json:"email"`
}

type json_struct struct {
	Hello string `json:"hello"`
}


// @Title Get User
// @Description Retrieves user
// @Success 200
// @Failure 400 Malformed request
// @router /users/0x:address[a-fA-F0-9] [get]
func (controller *UserController) Get() {
	var hashed_message []byte = crypto.Keccak256([]byte(controller.Ctx.Input.Param(":address")))
	var hex_string string = hex.EncodeToString(hashed_message)
	m := json_struct{hex_string}
	controller.Data["json"] = &m
	controller.ServeJSON()
}

// @Title Signup User
// @Description Registers user in the service for the kyc verification
// @Success 200
// @Failure 400 Malformed request
// @router /users/0x:address[a-fA-F0-9] [post]
func (controller *UserController) Post() {
	// Beego validator
	valid := validation.Validation{}

	// Validate address format and checksum
	// @TODO

	// Check if user already exists
	// @TODO

	// User doesn't exist, so we validate all params are compliant with the domain
	// Validate email
	logs.Info(fmt.Sprintf("%s", controller.Ctx.Input.RequestBody))
	var response UserPost
	err := json.Unmarshal(controller.Ctx.Input.RequestBody, &response)
	if err != nil {
		logs.Warn(err)
	}
	valid.Email(response.Email, "email")
	if valid.HasErrors() {
		// If there are error messages it means the validation didn't pass
		// Print error message
		controller.Data["json"] = &valid.Errors
		logs.Info(valid.Errors)
		controller.Abort("400")
		return 
	}
	// Check name and last name doesn't contain strange characters

	// Recover address based con signature and terms hash

	// Recovered address should be the same than the one used in the url


	var hashed_message []byte = crypto.Keccak256([]byte(controller.Ctx.Input.Param(":address")))
	var hex_string string = hex.EncodeToString(hashed_message)
	m := json_struct{hex_string}
	controller.Data["json"] = &m
	controller.ServeJSON()
}
