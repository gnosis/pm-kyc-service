package controllers

import "github.com/astaxie/beego"

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

type CreateOnfidoApplicant struct {
	Name     string `json:"first_name"`
	LastName string `json:"last_name"`
	Email    string `json:"email"`
}

type CreateSDKToken struct {
	Applicant string `json:"applicant_id"`
	Referrer  string `json:"referrer"`
}

type SDKToken struct {
	Token string `json:"token"`
}

type GetOnfidoApplicant struct {
	ID string `json:"id"`
}

type json_struct struct {
	Hello string `json:"hello"`
}

type ValidationError struct {
	Message string `json:"Message"`
	Key     string `json:"Key"`
}
