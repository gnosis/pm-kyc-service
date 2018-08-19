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

type json_struct struct {
	Hello string `json:"hello"`
}

type ValidationError struct {
	Message string `json:"Message"`
	Key     string `json:"Key"`
}
