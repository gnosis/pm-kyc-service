package controllers

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
	"github.com/astaxie/beego/validation"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/gnosis/pm-kyc-service/models"
)

// @Title Get User
// @Description Retrieves user
// @Success 200
// @Failure 400 Malformed request
// @router /users/0x:address[a-fA-F0-9] [get]
func (controller *UserController) Get() {

	o := orm.NewOrm()

	user := models.User{EthereumAddress: controller.Ctx.Input.Param(":address")}

	err := o.Read(&user)

	if err == nil {
		var hashed_message []byte = crypto.Keccak256([]byte(controller.Ctx.Input.Param(":address")))
		var hex_string string = hex.EncodeToString(hashed_message)
		m := json_struct{hex_string}
		controller.Ctx.Output.SetStatus(200)
		controller.Data["json"] = &m
		controller.ServeJSON()
	} else if err == orm.ErrNoRows {
		controller.Ctx.Output.SetStatus(404)
		controller.ServeJSON()
		return
	} else {
		controller.Abort("500")
		return
	}
}

// @Title Signup User
// @Description Registers user in the service for the kyc verification
// @Success 200
// @Failure 400 Malformed request
// @router /users/0x:address([a-fA-F0-9]+) [post]
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

	// Validate address format and checksum
	validChecksum := isValidChecksum(controller.Ctx.Input.Param(":address"))
	if !validChecksum {
		err := ValidationError{Message: "Invalid Checksum Address", Key: "address"}
		controller.Data["json"] = &err
		controller.Ctx.Output.SetStatus(400)
		controller.ServeJSON()
		return
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
	if len(request.Signature.TermsHash) > 2 && request.Signature.TermsHash[:2] == "0x" {
		request.Signature.TermsHash = request.Signature.TermsHash[2:]
	}
	valid.Length(request.Signature.TermsHash, 64, "terms hash")
	valid.Required(request.Signature.R, "r")
	valid.Numeric(request.Signature.R, "r")
	valid.Required(request.Signature.S, "s")
	valid.Numeric(request.Signature.S, "s")
	valid.Required(request.Signature.V, "v")
	valid.Numeric(request.Signature.V, "v")

	if valid.HasErrors() {
		// If there are error messages it means the validation didn't pass
		// Print error message
		controller.Data["json"] = &valid.Errors
		logs.Info(valid.Errors)
		controller.Ctx.Output.SetStatus(400)
		controller.ServeJSON()
		return
	}

	// Recover address based con signature and terms hash
	termsHash, err1 := hex.DecodeString(request.Signature.TermsHash)

	rInt, _ := (new(big.Int)).SetString(request.Signature.R, 10)

	sInt, _ := (new(big.Int)).SetString(request.Signature.S, 10)
	vInt, _ := strconv.Atoi(request.Signature.V)

	logs.Info("Signature", rInt, sInt, vInt)
	composedSignature := fmt.Sprintf("%064x%064x%02x", rInt, sInt, vInt-27)
	logs.Info("Composed signature (hex) ", composedSignature)

	signatureBytes, _ := hex.DecodeString(composedSignature)

	pubKey, err3 := secp256k1.RecoverPubkey(
		termsHash,
		signatureBytes,
	)

	if err1 != nil {
		logs.Warn(err1.Error())
	}
	if err3 != nil {
		logs.Warn(err3.Error())
	}
	logs.Info("pubkey", hex.EncodeToString(pubKey))
	recoveredAddress := hex.EncodeToString(crypto.Keccak256(pubKey[1:])[12:])
	logs.Info(recoveredAddress)

	// Recovered address should be the same than the one used in the url
	requestAddress := strings.ToLower(controller.Ctx.Input.Param(":address"))

	if requestAddress != recoveredAddress {
		err := ValidationError{Message: "Recovered address missmatch", Key: "address"}
		controller.Data["json"] = &err
		controller.Ctx.Output.SetStatus(401)
		controller.ServeJSON()
		return
	}

	// Check if user already exists
	o := orm.NewOrm()

	user := models.User{EthereumAddress: recoveredAddress}

	errRecover := o.Read(&user)

	if errRecover == nil {
		// User exists, we just generate the SDK token afterwards
		logs.Info("User: ", user.ApplicantID, user.TermsSignature)
		controller.Ctx.Output.SetStatus(200)
	} else if errRecover == orm.ErrNoRows {
		// User doesn't exists, so we save create an applicant on Onfido and save the model
		url := beego.AppConfig.String("apiURL") + "/applicants/"

		var onfidoData = CreateOnfidoApplicant{Name: request.Name, LastName: request.LastName, Email: request.Email}

		jsonData, _ := json.Marshal(onfidoData)

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", "Token token="+beego.AppConfig.String("apiToken"))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			logs.Error(err.Error())
			panic(err)
		}

		defer resp.Body.Close()

		logs.Info("response Status:", resp.Status)
		logs.Info("response Headers:", resp.Header)
		body, _ := ioutil.ReadAll(resp.Body)
		logs.Info("response Body:", string(body))
		var applicant GetOnfidoApplicant
		errJson := json.Unmarshal(body, &applicant)
		if errJson != nil {
			logs.Error(errJson.Error())
		}

		// TODO control error

		user.ApplicantID = applicant.ID
		user.TermsHash = request.Signature.TermsHash
		user.TermsSignature = composedSignature

		o.Insert(&user)
		controller.Ctx.Output.SetStatus(201)
	} else {
		logs.Error(errRecover.Error())
		controller.Abort("500")
		return
	}

	// Get SDK token

	url := beego.AppConfig.String("apiURL") + "/sdk_token/"

	var sdkData = CreateSDKToken{Applicant: user.ApplicantID, Referrer: "*://*/*"}

	jsonData, _ := json.Marshal(sdkData)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Token token="+beego.AppConfig.String("apiToken"))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logs.Error(err.Error())
		panic(err)
	}

	defer resp.Body.Close()

	logs.Info("response Status:", resp.Status)
	logs.Info("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	logs.Info("response Body:", string(body))
	var sdk SDKToken
	errJson := json.Unmarshal(body, &sdk)
	if errJson != nil {
		logs.Error(errJson.Error())
	}

	controller.Data["json"] = &sdk
	controller.ServeJSON()
}
