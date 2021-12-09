package model

import (
	"fmt"
	didTypes "github.com/allinbits/cosmos-cash/v2/x/did/types"
	vcTypes "github.com/allinbits/cosmos-cash/v2/x/verifiable-credential/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/hyperledger/aries-framework-go/pkg/doc/util"
	"github.com/hyperledger/aries-framework-go/pkg/doc/verifiable"
	"time"
)

// ChainAccountCredential creates a verifiable credential for a blockchain account
func ChainAccountCredential(chainID, address, did, name string) *verifiable.Credential {
	return &verifiable.Credential{
		Context: []string{
			"https://www.w3.org/2018/credentials/v1",
		},
		ID: didTypes.NewBlockchainAccountID(chainID, address).EncodeToString(),
		Types: []string{
			"VerifiableCredential",
			// "CosmosAccountAddressCredential",
		},
		Subject: map[string]string{
			"ID":      did,
			"Name":    name,
			"Address": address,
		},
		Issuer: verifiable.Issuer{
			ID: did,
		},
		Issued: util.NewTime(time.Now()),
	}
}

// NewPaymentReceiptCredential compose a payment receipt credential
func NewPaymentReceiptCredential(issuerDID, txHash string, pr PaymentRequest) *verifiable.Credential {
	return &verifiable.Credential{
		Context: []string{
			"https://www.w3.org/2018/credentials/v1",
		},
		ID: fmt.Sprint("cash:receipt#", txHash),
		Types: []string{
			"VerifiableCredential",
			// "PaymentReceiptCredential",
		},
		Subject: map[string]interface{}{
			"txHash":  txHash,
			"request": pr,
		},
		Issuer: verifiable.Issuer{
			ID: issuerDID,
		},
		Issued: util.NewTime(time.Now()),
	}
}

func NewRegulatorCredential(issuerDID, subjectDID string, cr RegulatorCredentialRequest) vcTypes.VerifiableCredential {
	return vcTypes.NewRegulatorVerifiableCredential(
		fmt.Sprint("regulator:", subjectDID),
		issuerDID,
		time.Now(),
		vcTypes.NewRegulatorCredentialSubject(
			subjectDID,
			cr.Name,
			cr.Country,
		),
	)
}

func NewRegistrationCredential(issuerDID, subjectDID string, cr RegistrationCredentialRequest) vcTypes.VerifiableCredential {
	return vcTypes.NewRegistrationVerifiableCredential(
		fmt.Sprint("registration:emti/", subjectDID),
		issuerDID,
		time.Now(),
		vcTypes.NewRegistrationCredentialSubject(
			subjectDID,
			cr.Country,
			cr.ShortName,
			cr.Name,
		),
	)
}

func NewLicenseCredential(issuerDID, subjectDID string, cr LicenseCredentialRequest) vcTypes.VerifiableCredential {
	return vcTypes.NewLicenseVerifiableCredential(
		fmt.Sprintf("license:%s/%s", cr.Denom, subjectDID),
		issuerDID,
		time.Now(),
		vcTypes.NewLicenseCredentialSubject(
			subjectDID,
			cr.LicenseType,
			cr.Country,
			cr.Authority,
			sdk.NewCoin(cr.Denom, sdk.NewInt(cr.MaxSupply)),
		),
	)
}

func NewPoKYCCredential(issuerDID, subjectDID string, cr EMoneyApplicationRequest) vcTypes.VerifiableCredential {
	return vcTypes.NewUserVerifiableCredential(
		fmt.Sprintf("PoKYC:%s", subjectDID),
		issuerDID,
		time.Now(),
		vcTypes.NewUserCredentialSubject(
			subjectDID,
			cr.ZKP,
			cr.IsVerified,
		),
	)
}
