package controllers

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
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
// @router /users/0x:address([a-fA-F0-9]+) [get]
func (controller *UserController) Get() {

	// Validate address format and checksum
	validChecksum := isValidChecksum(controller.Ctx.Input.Param(":address"))
	if !validChecksum {
		err := ValidationError{Message: "Invalid Checksum Address", Key: "address"}
		controller.Data["json"] = &err
		controller.Ctx.Output.SetStatus(400)
		controller.ServeJSON()
		return
	}

	o := orm.NewOrm()

	logs.Info("Getting user with address ", controller.Ctx.Input.Param(":address"))
	user := models.User{EthereumAddress: strings.ToLower(controller.Ctx.Input.Param(":address"))}

	err := o.Read(&user)

	if err == nil {
		// User exists, verify if it has checks
		var status bool
		if user.OnfidoCheck != nil {
			status = user.OnfidoCheck.IsVerified
		} else {
			status = false
		}

		response := UserStatus{status}

		controller.Ctx.Output.SetStatus(200)
		controller.Data["json"] = &response
		controller.ServeJSON()
		return
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

	// Recovered address should be the same than the one used in the reqURL
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
		reqURL := beego.AppConfig.String("apiURL") + "/applicants/"

		var onfidoData = CreateOnfidoApplicant{Name: request.Name, LastName: request.LastName, Email: request.Email}

		jsonData, _ := json.Marshal(onfidoData)

		req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(jsonData))
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

		if resp.Status != "201 Created" {
			controller.Ctx.Output.SetStatus(403)
			controller.ServeJSON()
			return
		}

		var applicant GetOnfidoApplicant
		errJson := json.Unmarshal(body, &applicant)
		if errJson != nil {
			logs.Error(errJson.Error())
		}

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

	reqURL := beego.AppConfig.String("apiURL") + "/sdk_token/"

	var sdkData = CreateSDKToken{Applicant: user.ApplicantID, Referrer: "*://*/*"}

	jsonData, _ := json.Marshal(sdkData)

	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(jsonData))
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

	if resp.Status != "200 OK" {
		controller.Ctx.Output.SetStatus(403)
		controller.ServeJSON()
		return
	}

	var sdk SDKToken
	errJson := json.Unmarshal(body, &sdk)
	if errJson != nil {
		logs.Error(errJson.Error())
	}

	controller.Data["json"] = &sdk
	controller.ServeJSON()
}

// @Title Signal User Report
// @Description After the user uploads the documents, the frontend reaches this endpoint to create the check
// @Success 200
// @Failure 400 Malformed request
// @router /users/0x:address([a-fA-F0-9]+) [put]
func (controller *UserController) Put() {

	// Validate address format and checksum
	validChecksum := isValidChecksum(controller.Ctx.Input.Param(":address"))
	if !validChecksum {
		err := ValidationError{Message: "Invalid Checksum Address", Key: "address"}
		controller.Data["json"] = &err
		controller.Ctx.Output.SetStatus(400)
		controller.ServeJSON()
		return
	}
	// Check if user exists
	o := orm.NewOrm()

	logs.Info("Getting user with address ", controller.Ctx.Input.Param(":address"))
	user := models.User{EthereumAddress: strings.ToLower(controller.Ctx.Input.Param(":address"))}

	err := o.Read(&user)

	if err == nil {
		// User exists
		// Verify if the check was already created
		_, errRelated := o.LoadRelated(&user, "OnfidoCheck")
		if errRelated != nil {
			logs.Info(errRelated.Error())
			controller.Ctx.Output.SetStatus(500)
			controller.ServeJSON()
			return
		}
		logs.Info("Check model: ", user.OnfidoCheck)
		if user.OnfidoCheck != nil {
			controller.Ctx.Output.SetStatus(204)
			controller.ServeJSON()
			return
		} else {
			// Create the check in Onfido
			reqURL := beego.AppConfig.String("apiURL") + "/applicants/" + user.ApplicantID + "/checks/"
			logs.Info("Creating check against ", reqURL, user.ApplicantID)

			form := url.Values{}
			form.Add("type", "standard")
			form.Add("reports[][name]", "identity")
			form.Add("reports[][name]", "document")
			form.Add("reports[][name]", "facial_similarity")

			req, err := http.NewRequest("POST", reqURL, strings.NewReader(form.Encode()))
			req.Header.Set("Authorization", "Token token="+beego.AppConfig.String("apiToken"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				logs.Error(err.Error())
				panic(err)
			}

			logs.Info("response Status:", resp.Status)
			logs.Info("response Headers:", resp.Header)
			body, _ := ioutil.ReadAll(resp.Body)
			logs.Info("response Body:", string(body))

			if resp.Status != "201 Created" {
				controller.Ctx.Output.SetStatus(403)
				controller.ServeJSON()
				return
			} else {
				defer resp.Body.Close()
				var checkResponse ResponseOnfidoCheck
				// Save check
				errJSON := json.Unmarshal(body, &checkResponse)
				if errJSON != nil {
					logs.Error(errJSON.Error())
				}
				check := models.OnfidoCheck{User: &user, CheckID: checkResponse.ID}
				insertID, insertErr := o.Insert(&check)
				if insertErr != nil || insertID == 0 {
					controller.Data["json"] = insertErr.Error()
					controller.Ctx.Output.SetStatus(500)
					controller.ServeJSON()
					return
				}
				logs.Info("Inserted ", insertID)

				controller.Ctx.Output.SetStatus(201)
				controller.ServeJSON()
				return
			}

		}
	} else if err == orm.ErrNoRows {
		controller.Ctx.Output.SetStatus(404)
		controller.ServeJSON()
		return
	} else {
		controller.Abort("500")
		return
	}
}

// @Title Liveness probe for Kubernetes
// @Description It just returns 200 if everything is ok, or 500 if the database is not connecting etc
// @Success 200
// @Failure 400 Malformed request
// @router /check [get]
func (controller *UserController) Check() {
	o := orm.NewOrm()
	var users []*models.User
	_, err := o.QueryTable("user").All(&users)

	if err != nil {
		controller.Ctx.Output.SetStatus(500)
		controller.ServeJSON()
		return
	} else {
		controller.Ctx.Output.SetStatus(200)
		controller.ServeJSON()
		return
	}
}
