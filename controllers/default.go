package controllers

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/validation"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
)

// Operations about users
type UserController struct {
	beego.Controller
}
type UserSignupSignature struct {
	TermsHash string `json:"termsHash"`
	R         string `json:"r"`
	S         string `json:"s"`
	V         string `json:"v"`
}
type UserPost struct {
	Email     string              `json:"email"`
	Name      string              `json:"name"`
	LastName  string              `json:"lastName"`
	Signature UserSignupSignature `json:"signature"`
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

	// Serialize json
	logs.Info(fmt.Sprintf("%s", controller.Ctx.Input.RequestBody))
	var request UserPost
	err := json.Unmarshal(controller.Ctx.Input.RequestBody, &request)
	if err != nil {
		logs.Warn(err)
	}

	// User doesn't exist, so we validate all params are compliant with the domain
	// Validate email
	valid.Required(request.Email, "email")
	valid.Email(request.Email, "email")
	// Check name and last name are included
	// We don't validate if it cointains strange characters
	valid.Required(request.Name, "name")
	valid.Required(request.LastName, "last name")

	valid.Required(request.Signature.TermsHash, "terms hash")
	valid.Required(request.Signature.R, "r")
	valid.Required(request.Signature.S, "s")
	valid.Required(request.Signature.V, "v")

	// Validate address format and checksum
	// @TODO

	// Check if user already exists
	// @TODO

	// Recover address based con signature and terms hash
	termsHash, err1 := hex.DecodeString("8144a6fa26be252b86456491fbcd43c1de7e022241845ffea1c3df066f7cfede")
	signatureBytes, err2 := hex.DecodeString("5043a71031083406de9a3686d0b3ea900add8a9ffb57a4b9c31b1611609a98f1294822c728fa3dcf61443956254db9f2fecc5c860b21635fcbd856f00e0552f700")
	pubKey, err3 := secp256k1.RecoverPubkey(
		termsHash,
		signatureBytes,
	)

	if err1 != nil {
		logs.Warn(err1.Error())
	}
	if err2 != nil {
		logs.Warn(err2.Error())
	}
	if err3 != nil {
		logs.Warn(err3.Error())
	}
	logs.Info("pubkey", hex.EncodeToString(pubKey))
	recoveredAddress := crypto.Keccak256(pubKey[1:])[12:]
	logs.Info(hex.EncodeToString(recoveredAddress))

	// Recovered address should be the same than the one used in the url

	if valid.HasErrors() {
		// If there are error messages it means the validation didn't pass
		// Print error message
		controller.Data["json"] = &valid.Errors
		logs.Info(valid.Errors)
		controller.Ctx.Output.SetStatus(400)
		controller.ServeJSON()
		return
	}
	/*
		var hashed_message []byte = crypto.Keccak256([]byte(controller.Ctx.Input.Param(":address")))
		var hex_string string = hex.EncodeToString(hashed_message)
		m := json_struct{hex_string}
	*/
	controller.Data["json"] = &request
	controller.ServeJSON()
}
