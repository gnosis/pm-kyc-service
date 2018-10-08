package controllers

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
	"github.com/astaxie/beego/validation"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gnosis/pm-kyc-service/contracts"
	"github.com/gnosis/pm-kyc-service/models"
	"github.com/onrik/ethrpc"
)

// @Title Get User
// @Description Retrieves user
// @Success 200
// @Failure 400 Malformed request
// @router /users/0x:address([a-fA-F0-9]+) [get]
func (controller *UserController) Get() {

	// Validate address format and checksum
	userAddress := controller.Ctx.Input.Param(":address")
	validChecksum := isValidChecksum(userAddress)
	if !validChecksum {
		err := ValidationError{Message: "Invalid Checksum Address", Key: "address"}
		controller.Data["json"] = &err
		controller.Ctx.Output.SetStatus(400)
		controller.ServeJSON()
		return
	}
	// Check if user exists
	o := orm.NewOrm()

	logs.Info("Getting user with address ", userAddress)
	user := models.OnfidoUser{EthereumAddress: strings.ToLower(userAddress)}
	err := o.Read(&user)
	o.LoadRelated(&user, "OnfidoCheck")

	if err == nil {
		// User exists, verify if it has checks
		var status OnfidoStatus
		if user.OnfidoCheck == nil {
			status = PENDING_DOCUMENT_UPLOAD
		} else {
			if user.OnfidoCheck.IsVerified {
				if user.OnfidoCheck.IsClear {
					status = ACCEPTED
				} else {
					status = DENIED
				}
			} else {
				status = WAITING_FOR_APPROVAL
			}
		}

		response := UserStatus{status.String()}

		controller.Ctx.Output.SetStatus(200)
		controller.Data["json"] = &response
		controller.ServeJSON()
		return
	} else if err == orm.ErrNoRows {
		controller.Ctx.Output.SetStatus(404)
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
	ethereumAddress := controller.Ctx.Input.Param(":address")
	validChecksum := isValidChecksum(ethereumAddress)
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
	valid.Required(request.Signature.Terms, "terms")

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

	calculatedTermsHash := hex.EncodeToString(crypto.Keccak256([]byte(request.Signature.Terms)))

	if calculatedTermsHash != request.Signature.TermsHash {
		message := fmt.Sprintf("Terms calculated hash %s mismatch with termsHash %s", calculatedTermsHash, request.Signature.TermsHash)
		err := ValidationError{Message: message, Key: "terms"}
		controller.Data["json"] = &err
		controller.Ctx.Output.SetStatus(400)
		controller.ServeJSON()
		return
	}

	// Check eth account has balance
	ethereumRPCURL := beego.AppConfig.String("ethereumRPCURL")
	rpc := ethrpc.NewEthRPC(ethereumRPCURL)
	logs.Info("ETH RPC Connection %s", ethereumRPCURL)
	balance, err := rpc.EthGetBalance("0x"+ethereumAddress, "latest")
	if err != nil {
		logs.Error(err)
		err := ValidationError{Message: "Error recovering balance from address", Key: "address"}
		controller.Data["json"] = &err
		controller.Ctx.Output.SetStatus(500)
		controller.ServeJSON()
		return
	}

	minimumBalanceWei, _ := (new(big.Int)).SetString(beego.AppConfig.String("minimumBalanceWei"), 10)

	if balance.Cmp(minimumBalanceWei) == -1 {
		message := fmt.Sprintf("Balance for account %s should be at least %s wei and is %s wei", ethereumAddress, minimumBalanceWei.String(), balance.String())
		err := ValidationError{Message: message, Key: "address"}
		controller.Data["json"] = &err
		controller.Ctx.Output.SetStatus(500)
		controller.ServeJSON()
		return
	}

	// Recover address based on signature and terms hash
	termsHash, err1 := hex.DecodeString(request.Signature.TermsHash)

	rInt, _ := (new(big.Int)).SetString(request.Signature.R, 10)
	sInt, _ := (new(big.Int)).SetString(request.Signature.S, 10)
	vInt, _ := strconv.Atoi(request.Signature.V)

	requestAddress := strings.ToLower(ethereumAddress)
	logs.Info("Signature", rInt, sInt, vInt)
	composedSignature := fmt.Sprintf("%064x%064x%02x", rInt, sInt, vInt-27)
	logs.Info("Composed signature (hex) ", composedSignature)

	if vInt == 1 {
		//Contract signature
		contractAddress := common.HexToAddress(ethereumAddress)
		client, err := ethclient.Dial(ethereumRPCURL)
		if err != nil {
			logs.Error("Unable to connect to ethereum network: %v", err)
		}
		instance, err := contracts.NewISignatureValidator(contractAddress, client)
		if err != nil {
			logs.Error("Unable to connect to contract: %s", ethereumAddress)
		}
		// TODO Add terms to signature
		valid, err := instance.IsValidSignature(nil, []byte(request.Signature.Terms), nil)
		if err != nil {
			logs.Error(err)
			err := ValidationError{Message: "Cannot check if signature is valid on contract", Key: "address"}
			controller.Data["json"] = &err
			controller.Ctx.Output.SetStatus(500)
			controller.ServeJSON()
			return
		}

		if !valid {
			message := fmt.Sprintf("Signature for terms \"%s\" not valid on contract %s", request.Signature.Terms, ethereumAddress)
			logs.Info(message)
			err := ValidationError{Message: message, Key: "address"}
			controller.Data["json"] = &err
			controller.Ctx.Output.SetStatus(400)
			controller.ServeJSON()
			return
		}
	} else {
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
		if requestAddress != recoveredAddress {
			message := fmt.Sprintf("Recovered address is %s: missmatch", recoveredAddress)
			err := ValidationError{Message: message, Key: "address"}
			controller.Data["json"] = &err
			controller.Ctx.Output.SetStatus(401)
			controller.ServeJSON()
			return
		}
	}

	// Check if user already exists
	o := orm.NewOrm()

	user := models.OnfidoUser{EthereumAddress: requestAddress}

	errRecover := o.Read(&user)

	if errRecover == nil {
		// User exists, we just generate the SDK token afterwards
		logs.Info("User: ", user.ApplicantId, user.TermsSignature)
		controller.Ctx.Output.SetStatus(200)
	} else if errRecover == orm.ErrNoRows {
		// User doesn't exists, so we save create an applicant on Onfido and save the model

		onfidoApplicant := CreateOnfidoApplicant(request.Name, request.LastName, request.Email)
		if onfidoApplicant == nil {
			controller.Ctx.Output.SetStatus(403)
			return
		}
		user.ApplicantId = onfidoApplicant.ID
		user.TermsHash = request.Signature.TermsHash
		user.TermsSignature = composedSignature

		insertID, insertErr := o.Insert(&user)
		if insertErr != nil {
			logs.Error(insertErr.Error())
		}
		logs.Info(insertID)
		controller.Ctx.Output.SetStatus(201)
	} else {
		logs.Error(errRecover.Error())
		controller.Abort("500")
		return
	}

	// Get SDK token
	onfidoSDKToken := GetOnfidoSDKToken(user.ApplicantId)
	if onfidoSDKToken == nil {
		controller.Ctx.Output.SetStatus(403)
		return
	}

	controller.Data["json"] = onfidoSDKToken
	controller.ServeJSON()
}

// @Title Signal User Report
// @Description After the user uploads the documents, the frontend reaches this endpoint to create the check
// @Success 200
// @Failure 400 Malformed request
// @router /users/0x:address([a-fA-F0-9]+) [put]
func (controller *UserController) Put() {

	// Validate address format and checksum
	userAddress := controller.Ctx.Input.Param(":address")
	validChecksum := isValidChecksum(userAddress)
	if !validChecksum {
		err := ValidationError{Message: "Invalid Checksum Address", Key: "address"}
		controller.Data["json"] = &err
		controller.Ctx.Output.SetStatus(400)
		controller.ServeJSON()
		return
	}
	// Check if user exists
	o := orm.NewOrm()

	logs.Info("Getting user with address ", userAddress)
	user := models.OnfidoUser{EthereumAddress: strings.ToLower(userAddress)}

	err := o.Read(&user)

	if err == nil {
		// User exists
		// Verify if the check was already created
		o.LoadRelated(&user, "OnfidoCheck")
		// LoadRelated returns `<QuerySeter> no row found` if object does not exists
		/* _, errRelated := o.LoadRelated(&user, "OnfidoCheck")
		if errRelated != nil {
			logs.Info("Cannot find OnfidoCheck for user with address ", userAddress)
			controller.Ctx.Output.SetStatus(500)
			controller.ServeJSON()
			return
		}*/
		logs.Info("OnfidoCheck model: ", user.OnfidoCheck)
		if user.OnfidoCheck != nil {
			controller.Ctx.Output.SetStatus(204)
			return
		}

		onfidoCheck := CreateOnfidoCheck(user.ApplicantId)
		if onfidoCheck == nil {
			controller.Ctx.Output.SetStatus(403)
			return
		}

		onfidoCheckModel := models.OnfidoCheck{User: &user, CheckId: onfidoCheck.ID}
		insertID, insertErr := o.Insert(&onfidoCheckModel)
		if insertErr != nil {
			logs.Error(insertErr.Error())
		}
		logs.Info("Inserted ", insertID)

		controller.Ctx.Output.SetStatus(201)
		return
	} else if err == orm.ErrNoRows {
		controller.Ctx.Output.SetStatus(404)
		return
	} else {
		controller.Abort("500")
		return
	}
}

// @Title Webhook post
// @Description Called by Onfido when report status changes
// @Success 200
// @Failure 400 Malformed request
// @router /webhooks [post]
func (controller *UserController) WebhookPost() {
	webhookToken := beego.AppConfig.String("webhookToken")
	signature := controller.Ctx.Input.Header("X-Signature")
	signatureHex, _ := hex.DecodeString(signature)

	logs.Info(fmt.Sprintf("Received WebHook with signature %s and content %s", signature, controller.Ctx.Input.RequestBody))

	if CheckHmac(controller.Ctx.Input.RequestBody, signatureHex, []byte(webhookToken)) == false {
		logs.Warn(fmt.Sprintf("Invalid signature %s using webhookToken %s", signature, webhookToken))
		controller.Ctx.Output.SetStatus(400)
		return
	}

	// Serialize json
	var request OnfidoWebHook

	err := json.Unmarshal(controller.Ctx.Input.RequestBody, &request)
	if err != nil {
		logs.Warn(err)
		controller.Ctx.Output.SetStatus(400)
		return
	}

	if request.IsReportCompleted() == false {
		logs.Warn("Request not matching action. Payload action is", request.Payload.Action)
		controller.Ctx.Output.SetStatus(200)
		return
	}

	// Check if user already exists
	o := orm.NewOrm()

	onfidoCheck := models.OnfidoCheck{CheckId: request.Payload.Object.Id}

	if o.Read(&onfidoCheck) == nil {
		// Load Related is not working
		// o.LoadRelated(&onfidoCheck, "User")
		user := models.OnfidoUser{EthereumAddress: onfidoCheck.User.EthereumAddress}
		o.Read(&user)
		onfidoAPICheck := GetOnfidoCheck(user.ApplicantId, onfidoCheck.CheckId)
		logs.Info("Onfido report completed for id", request.Payload.Object.Id, "with result", onfidoAPICheck.Result)
		onfidoCheck.IsVerified = true
		onfidoCheck.IsClear = onfidoAPICheck.IsClear()
		o.Update(&onfidoCheck, "IsVerified", "IsClear")
		controller.Ctx.Output.SetStatus(200)
		return
	}
	controller.Ctx.Output.SetStatus(404)
	return
}

// @Title Mark user as approved, testing purposes only
// @Description Mark user as approved. It's GET to allow not technical people to trigger it
// @Success 200
// @Failure 400 Malformed request
// @router /users/approval/0x:address([a-fA-F0-9]+) [get]
func (controller *UserController) ApproveUser() {
	manualUserApproval := beego.AppConfig.DefaultBool("manualUserApproval", false)

	if !manualUserApproval {
		err := ValidationError{Message: "Manual user approval not enabled", Key: "manualUserApproval"}
		controller.Data["json"] = &err
		controller.Ctx.Output.SetStatus(403)
		controller.ServeJSON()
		return
	}

	// Validate address format and checksum
	userAddress := controller.Ctx.Input.Param(":address")

	// Check if user exists
	o := orm.NewOrm()

	logs.Info("Getting user with address ", userAddress)
	user := models.OnfidoUser{EthereumAddress: strings.ToLower(userAddress)}
	err := o.Read(&user)

	if err == nil {
		onfidoCheck := user.OnfidoCheck
		onfidoCheck.IsVerified = true
		onfidoCheck.IsClear = true
		o.Update(&onfidoCheck, "IsVerified", "IsClear")
		userStatus := UserStatus{Status: "User Verified"}
		controller.Data["json"] = &userStatus
		controller.Ctx.Output.SetStatus(200)
		controller.ServeJSON()
		return
	} else if err == orm.ErrNoRows {
		controller.Ctx.Output.SetStatus(404)
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
	var users []*models.OnfidoUser
	_, err := o.QueryTable("onfido_user").All(&users)

	if err != nil && err != orm.ErrNoRows {
		controller.Ctx.Output.SetStatus(500)
		return
	} else {
		controller.Ctx.Output.SetStatus(200)
		return
	}
}
