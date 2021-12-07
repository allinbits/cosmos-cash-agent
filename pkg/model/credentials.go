package model

import (
	"fmt"
	didTypes "github.com/allinbits/cosmos-cash/v2/x/did/types"
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
