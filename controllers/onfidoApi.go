package controllers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
)

func CreateOnfidoApplicant(name, lastName, email string) *OnfidoApplicant {
	reqURL := beego.AppConfig.String("apiURL") + "/applicants/"

	var onfidoData = OnfidoApplicantCreation{Name: name, LastName: lastName, Email: email}

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
		return nil
	}

	var applicant OnfidoApplicant
	if errJSON := json.Unmarshal(body, &applicant); errJSON != nil {
		logs.Error(errJSON.Error())
	}
	return &applicant
}

func CreateOnfidoCheck(applicantId string) *OnfidoCheck {
	// Create the check in Onfido
	reqURL := beego.AppConfig.String("apiURL") + "/applicants/" + applicantId + "/checks/"
	logs.Info("Creating check against ", reqURL, applicantId)

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

	defer resp.Body.Close()
	if resp.Status != "201 Created" {
		return nil
	}

	var OnfidoCheck OnfidoCheck
	// Save check
	errJSON := json.Unmarshal(body, &OnfidoCheck)
	if errJSON != nil {
		logs.Error(errJSON.Error())
	}
	return &OnfidoCheck
}

func GetOnfidoCheck(applicantId, checkId string) OnfidoCheck {
	apiToken := beego.AppConfig.String("apiToken")
	reqURL := beego.AppConfig.String("apiURL") + "/applicants/" + applicantId + "/checks/" + checkId

	req, err := http.NewRequest("GET", reqURL, nil)
	req.Header.Set("Authorization", "Token token="+apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logs.Error(err.Error())
		panic(err)
	}

	defer resp.Body.Close()

	logs.Info("Getting Onfido Check From Api. Status:", resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)
	logs.Info("Response Body:", string(body))

	var onfidoCheck OnfidoCheck
	errJSON := json.Unmarshal(body, &onfidoCheck)
	if errJSON != nil {
		logs.Error(errJSON.Error())
	}

	return onfidoCheck
}

func GetOnfidoSDKToken(applicantId string) *SDKToken {
	var sdkData = CreateSDKToken{Applicant: applicantId, Referrer: "*://*/*"}
	reqURL := beego.AppConfig.String("apiURL") + "/sdk_token/"

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
		return nil
	}

	var sdk SDKToken
	if errJSON := json.Unmarshal(body, &sdk); errJSON != nil {
		logs.Error(errJSON.Error())
	}
	return &sdk

}
