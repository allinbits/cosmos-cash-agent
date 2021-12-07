package model

import (
	"encoding/json"
	"strings"
)

// PresentationRequest represent a verifiable credential schema
type PresentationRequest interface {
	ExpectedCredential() string
}

// ParsePresentationRequest try to parse a string to a presentation
func ParsePresentationRequest(data string) (isPresentation bool, v interface{}) {
	d := json.NewDecoder(strings.NewReader(data))
	d.DisallowUnknownFields()

	options := []PresentationRequest{
		&PaymentRequest{},
	}
	// for each know interface try to parse it
	for _, into := range options {
		if err := d.Decode(into); err == nil {
			isPresentation, v = true, into
			return
		}
	}
	return
}

// In this section we define the concrete types for presentation requests
// tag string:  cash_label:"", cash_hint:""

type PaymentRequest struct {
	// Credential name of the credential that should be received as a reply for this request
	Credential string  `json:"expected_credential"`
	Denom      string  `json:"denom" cash_label:"Token Denomination" cash_hint:"the token symbol requested by the recipient"`
	Recipient  string  `json:"recipient" cash_label:"Recipient Address" cash_hint:"address to send the payment to"`
	Amount     float64 `json:"amount" cash_label:"Amount" cash_hint:"amount to be transferred"`
	Note       string  `json:"note" cash_label:"Payment Note" cash_hint:"note for the payment request"`
}

func NewPaymentRequest(denom, note string) PaymentRequest {
	return PaymentRequest{
		Denom:      denom,
		Note:       note,
		Credential: "PaymentReceiptCredential",
	}
}

func (r PaymentRequest) ExpectedCredential() string {
	return r.Credential
}
