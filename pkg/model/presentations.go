package model

import (
	"github.com/hyperledger/aries-framework-go/pkg/doc/verifiable"
	log "github.com/sirupsen/logrus"
)

// SchemaField is a field within the credential schema
type SchemaField struct {
	Name        string
	Title       string
	Description string
	Value       string
	ReadOnly    bool
}

func NewSchemaField(name, title, description string) SchemaField {
	return SchemaField{
		Name:        name,
		Title:       title,
		Description: description,
		ReadOnly:    false,
	}
}
func NewSchemaLabel(name, title, description, value string) SchemaField {
	return SchemaField{
		Name:        name,
		Title:       title,
		Description: description,
		Value:       value,
		ReadOnly:    true,
	}
}

// PresentationRequest represent a verifiable credential schema
type PresentationRequest struct {
	// The name of the presentation request
	Name string
	// Credential Should match the Verifiable Credentials that belongs to
	Credential string
	// Describe the request
	Fields []SchemaField
}

// LicenseSchema returns a license schema
func LicenseSchema(licenseType, authority string) PresentationRequest {
	//"license_type": "MICAEMI",
	//"country": "EU",
	//"authority": "Another Financial Services Body (AFFB)",
	//"circulation_limit": {
	//	"denom": "sEUR",
	//	"amount": "1000000000"
	//}
	return PresentationRequest{
		Name:       "LicensePresentationRequest",
		Credential: "LicenseCredential",
		Fields: []SchemaField{
			NewSchemaLabel("license_type", "License Type", "license acronym", licenseType),
			NewSchemaLabel("authority", "Authority", "authority issuing the license", authority),
			NewSchemaField("country", "Country", "country where the license applies"),
			NewSchemaField("denom", "Token denomination", "the token symbol"),
			NewSchemaField("amount", "Amount", "amount approved by the license"),
		},
	}
}

func PaymentRequest(address, denom, reason string) PresentationRequest {
	return PresentationRequest{
		Name:       "PaymentPresentationRequest",
		Credential: "ReceiptCredential",
		Fields: []SchemaField{
			NewSchemaLabel("reason", "Payment reason", "reason for the payment request", address),
			NewSchemaLabel("recipient_address", "Recipient Address", "address to send the payment to", address),
			NewSchemaLabel("denom", "Token denomination", "the token symbol requested by the recipient", denom),
			NewSchemaField("amount", "Amount", "payment request amount"),
		},
	}
}

func LicensePresentation(license *verifiable.Credential) {
	vp, err := verifiable.NewPresentation(verifiable.WithCredentials(license))
	if err != nil {
		log.Errorln(err)
	}
	log.Debugln(vp)
}
