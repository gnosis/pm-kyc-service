package controllers

import "github.com/astaxie/beego"

type OnfidoStatus int

const (
	PENDING_DOCUMENT_UPLOAD OnfidoStatus = iota
	WAITING_FOR_APPROVAL
	ACCEPTED
	DENIED
)

func (this OnfidoStatus) String() string {
	names := [...]string{
		"PENDING_DOCUMENT_UPLOAD",
		"WAITING_FOR_APPROVAL",
		"ACCEPTED",
		"DENIED"}
	if this < PENDING_DOCUMENT_UPLOAD || this > DENIED {
		return "Unknown"
	}
	return names[this]
}

// Operations about users
type UserController struct {
	beego.Controller
}

type UserSignupSignature struct {
	Terms     string `json:"terms"`
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

type OnfidoApplicantCreation struct {
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

type ValidationError struct {
	Message string `json:"Message"`
	Key     string `json:"Key"`
}

type UserStatus struct {
	Status string `json:"status"`
}

type Report struct {
	Name string `json:"name"`
}

type OnfidoApplicant struct {
	ID string `json:"id"`
}

type OnfidoCheckCreation struct {
	Reports []Report `json:"reports"`
}

// https://documentation.onfido.com/#check-object
type OnfidoCheck struct {
	ID     string `json:"id"`
	Result string `json:"result"`
}

func (this *OnfidoCheck) IsClear() bool {
	return this.Result == "clear"
}

type OnfidoWebHook struct {
	Payload OnfidoPayload `json:"payload"`
}

func (this *OnfidoWebHook) IsReportCompleted() bool {
	return this.Payload.Action == "report.completed"
}

type OnfidoPayload struct {
	Action       string       `json:"action"`
	ResourceType string       `json:"resource_type"`
	Object       OnfidoObject `json:"object"`
}

type OnfidoObject struct {
	CompletedAt string `json:"completed_at"`
	Href        string `json:"href"`
	Id          string `json:"id"`
	Status      string `json:"status"`
}
