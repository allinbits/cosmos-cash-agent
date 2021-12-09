package model

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"strings"
)

// PresentationRequest represent a verifiable credential schema
type PresentationRequest interface {
	ExpectedCredential() string
}

// ParsePresentationRequest try to parse a string to a presentation
func ParsePresentationRequest(data string) (v PresentationRequest, isPresentation bool) {
	d := json.NewDecoder(strings.NewReader(data))
	d.DisallowUnknownFields()

	options := []PresentationRequest{
		&PaymentRequest{},
		&RegulatorCredentialRequest{},
		&RegistrationCredentialRequest{},
		&LicenseCredentialRequest{},
		&EMoneyApplicationRequest{},
	}
	// for each know interface try to parse it
	for _, into := range options {
		if err := d.Decode(into); err == nil {
			isPresentation, v = true, into
			return
		} else {
			log.Errorln("error decoding", err)
		}
	}
	return
}

// In this section we define the concrete types for presentation requests
// tag string:  cash_label:"", cash_hint:""

type PaymentRequest struct {
	// Credential name of the credential that should be received as a reply for this request
	Credential string  `json:"expected_credential"`
	Recipient  string  `json:"recipient" cash_label:"Recipient Address" cash_hint:"address to send the payment to"`
	Amount     float64 `json:"amount" cash_label:"Amount" cash_hint:"amount to be transferred"`
	Denom      string  `json:"denom" cash_label:"Token Denomination" cash_hint:"the token symbol"`
	Note       string  `json:"note" cash_label:"Payment Note" cash_hint:"note for the payment request"`
}

func (r PaymentRequest) ExpectedCredential() string {
	return r.Credential
}

func NewPaymentRequest(denom, note string) PaymentRequest {
	return PaymentRequest{
		Denom:      denom,
		Note:       note,
		Credential: "PaymentReceiptCredential",
	}
}

// RegulatorCredentialRequest to activate a regulator
type RegulatorCredentialRequest struct {
	// Credential name of the credential that should be received as a reply for this request
	Credential string `json:"expected_credential"`
	SubjectDID string `json:"subject_did" cash_label:"Subject DID" cash_hint:"the subject that should be a regulator"`
	Country    string `json:"country" cash_label:"Country" cash_hint:"the regulator country of authority"`
	Name       string `json:"name" cash_label:"Regulator name" cash_hint:"name of the regulator authority"`
}

func (r RegulatorCredentialRequest) ExpectedCredential() string {
	return r.Credential
}

func NewRegulatorCredentialRequest(subjectDID string) RegulatorCredentialRequest {
	return RegulatorCredentialRequest{
		SubjectDID: subjectDID,
		Credential: "RegulatorCredential",
	}
}

// RegistrationCredentialRequest to register an emti organization
type RegistrationCredentialRequest struct {
	// Credential name of the credential that should be received as a reply for this request
	Credential string `json:"expected_credential"`
	Country    string `json:"country" cash_label:"Country" cash_hint:"the country of operation for the e-money token provider"`
	Name       string `json:"name" cash_label:"Organization name" cash_hint:"name of the e-money token provider organization"`
	ShortName  string `json:"name" cash_label:"Organization short name" cash_hint:"short name of the e-money token provider organization"`
}

func (r RegistrationCredentialRequest) ExpectedCredential() string {
	return r.Credential
}

func NewRegistrationCredentialRequest(country string) RegistrationCredentialRequest {
	return RegistrationCredentialRequest{
		Country:    country,
		Credential: "RegistrationCredential",
	}
}

// LicenseCredentialRequest to provide a license for an emti
type LicenseCredentialRequest struct {
	// Credential name of the credential that should be received as a reply for this request
	Credential  string `json:"expected_credential"`
	Country     string `json:"country" cash_label:"Country" cash_hint:"the regulator country of authority"`
	LicenseType string `json:"name" cash_label:"Regulator name" cash_hint:"name of the regulator authority"`
	Denom       string `json:"denom" cash_label:"Token denomination" cash_hint:"symbol of the token to issue"`
	MaxSupply   int64  `json:"max_supply" cash_label:"Max supply" cash_hint:"max supply for the token"`
	Authority   string `json:"authority" cash_label:"Regulator name" cash_hint:"name of the regulator authority"`
}

func (r LicenseCredentialRequest) ExpectedCredential() string {
	return r.Credential
}

func NewLicenseCredentialRequest(licenseType, country string) LicenseCredentialRequest {
	return LicenseCredentialRequest{
		Country:     country,
		LicenseType: licenseType,
		Credential:  "LicenseCredential",
	}
}

// EMoneyApplicationRequest to create a proof of KYC
type EMoneyApplicationRequest struct {
	// Credential name of the credential that should be received as a reply for this request
	Credential string `json:"expected_credential"`
	Name       string `json:"-" cash_label:"Name" cash_hint:"name of the applicant"`
	Surname    string `json:"-" cash_label:"Surname" cash_hint:"surname of the applicant"`
	Age        string `json:"-" cash_label:"Age" cash_hint:"age of the applicant"`
	ZKP        string `json:"zkp" cash_label:"ZKP" cash_hint:"autogenerated proof of the user identity (PoKYC) "`
	Amount     int64  `json:"amount" cash_label:"Requested amount" cash_hint:"amount requested in the e-money application"`
	IsVerified bool   `json:"is_verified"`
}

func (r EMoneyApplicationRequest) ExpectedCredential() string {
	return r.Credential
}

func NewEMoneyApplicationRequest() EMoneyApplicationRequest {
	return EMoneyApplicationRequest{
		IsVerified: true,
		Credential: "RegulatorCredential",
	}
}
