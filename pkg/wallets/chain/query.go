package chain

import (
	"context"
	didTypes "github.com/allinbits/cosmos-cash/v2/x/did/types"
	vcTypes "github.com/allinbits/cosmos-cash/v2/x/verifiable-credential/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	log "github.com/sirupsen/logrus"
)

// GetBalance retrieves the "cash" balance for an account
func (cc *ChainClient) GetBalance(address string) *sdk.Coin {
	bankClient := banktypes.NewQueryClient(cc.cos.Context)
	bankRes, err := bankClient.Balance(
		context.Background(),
		&banktypes.QueryBalanceRequest{Address: address, Denom: "cash"},
	)
	if err != nil {
		log.Fatalln("error requesting balance", err)
	}
	log.Infoln("balance for", address, "is", bankRes.GetBalance())
	return bankRes.GetBalance()
}

// GetBalances retrieves all the balances for an account
func (cc *ChainClient) GetBalances(address string) sdk.Coins {
	bankClient := banktypes.NewQueryClient(cc.cos.Context)
	bankRes, err := bankClient.AllBalances(
		context.Background(),
		&banktypes.QueryAllBalancesRequest{Address: address},
	)
	if err != nil {
		log.Fatalln("error requesting balance", err)
	}
	log.Infoln("balances for", address, "are", bankRes.GetBalances())
	return bankRes.GetBalances()
}

// GetChainOfTrust retrieve the chain of trust for a token DENOM
func (cc *ChainClient) GetChainOfTrust(licenseCredentialID string) (cot []vcTypes.VerifiableCredential) {
	client := vcTypes.NewQueryClient(cc.cos.Context)
	res, err := client.VerifiableCredentials(
		context.Background(),
		&vcTypes.QueryVerifiableCredentialsRequest{},
	)
	if err != nil {
		log.Fatalln("error requesting balance", err)
	}

	// first search for the license
	var issuerDID didTypes.DID
	for _, v := range res.GetVcs() {
		switch s := v.CredentialSubject.(type) {
		case *vcTypes.VerifiableCredential_LicenseCred:
			if s.LicenseCred.Id == licenseCredentialID {
				cot = append(cot, v)
				issuerDID = v.GetSubjectDID()
				break
			}
		}
	}
	// if the license is not found then return
	if issuerDID.String() == "" {
		return
	}

	// search for the issuer registartion credential
	var regulatorDID didTypes.DID
	for _, v := range res.GetVcs() {
		switch v.CredentialSubject.(type) {
		case *vcTypes.VerifiableCredential_RegistrationCred:
			if v.GetSubjectDID() == issuerDID {
				cot = append(cot, v)
				regulatorDID = v.GetIssuerDID()
				break
			}
		}
	}

	if regulatorDID.String() == "" {
		return
	}

	// search for the regulator credentials
	for _, v := range res.GetVcs() {
		switch v.CredentialSubject.(type) {
		case *vcTypes.VerifiableCredential_RegulatorCred:
			if v.GetSubjectDID() == regulatorDID {
				cot = append(cot, v)
				regulatorDID = v.GetIssuerDID()
				break
			}
		}
	}
	return
}



// GetDenomChainOfTrust retrieve the chain of trust for a token DENOM
func (cc *ChainClient) GetDenomChainOfTrust(denom string) (cot []vcTypes.VerifiableCredential) {
	client := vcTypes.NewQueryClient(cc.cos.Context)
	res, err := client.VerifiableCredentials(
		context.Background(),
		&vcTypes.QueryVerifiableCredentialsRequest{},
	)
	if err != nil {
		log.Fatalln("error requesting balance", err)
	}

	// first search for the license
	var issuerDID didTypes.DID
	for _, v := range res.GetVcs() {
		if s, ok := v.CredentialSubject.(*vcTypes.VerifiableCredential_LicenseCred); ok {
			if s.LicenseCred.CirculationLimit.GetDenom() == denom {
				cot = append(cot, v)
				issuerDID = v.GetSubjectDID()
				break
			}
		}
	}
	// if the license is not found then return
	if issuerDID.String() == "" {
		return
	}

	// search for the issuer registartion credential
	var regulatorDID didTypes.DID
	for _, v := range res.GetVcs() {
		if _, ok := v.CredentialSubject.(*vcTypes.VerifiableCredential_RegistrationCred); ok {
			if v.GetSubjectDID() == issuerDID {
				cot = append(cot, v)
				regulatorDID = v.GetIssuerDID()
				break
			}
		}
	}

	if regulatorDID.String() == "" {
		return
	}

	// search for the regulator credentials
	for _, v := range res.GetVcs() {
		if _, ok := v.CredentialSubject.(*vcTypes.VerifiableCredential_RegulatorCred); ok {
			if v.GetSubjectDID() == regulatorDID {
				cot = append(cot, v)
				regulatorDID = v.GetIssuerDID()
				break
			}
		}
	}
	return
}

// DIDDoc retrieve a did document for a
func (cc *ChainClient) DIDDoc(didID string) didTypes.DidDocument {
	client := didTypes.NewQueryClient(cc.cos.Context)
	res, err := client.DidDocument(context.Background(), &didTypes.QueryDidDocumentRequest{Id: didID})
	if err != nil {
		log.Fatalln("error requesting balance", err)
	}
	log.Infoln("did document for", didID, "is", res.GetDidDocument())
	return res.GetDidDocument()
}

// GetHolderPublicVCS retrieve the VCS holded by a did
func (cc *ChainClient) GetHolderPublicVCS(didID string) (vcs []vcTypes.VerifiableCredential) {
	client := vcTypes.NewQueryClient(cc.cos.Context)
	res, err := client.VerifiableCredentials(
		context.Background(),
		&vcTypes.QueryVerifiableCredentialsRequest{},
	)
	if err != nil {
		log.Fatalln("error requesting balance", err)
	}
	holderDID := didTypes.DID(didID)
	for _, v := range res.GetVcs() {
		if v.GetSubjectDID() == holderDID {
			vcs = append(vcs, v)
		}
	}
	return
}

// GetLicenseCredentials retrieve the VCS holded by a did
func (cc *ChainClient) GetLicenseCredentials() (vcs []vcTypes.VerifiableCredential) {
	client := vcTypes.NewQueryClient(cc.cos.Context)
	res, err := client.VerifiableCredentials(
		context.Background(),
		&vcTypes.QueryVerifiableCredentialsRequest{},
	)
	if err != nil {
		log.Fatalln("error requesting balance", err)
	}
	for _, v := range res.GetVcs() {
		if _, ok := v.CredentialSubject.(*vcTypes.VerifiableCredential_LicenseCred); ok {
			vcs = append(vcs, v)
		}
	}
	return
}

// GetPublicVC retrieve a vc by id
func (cc *ChainClient) GetPublicVC(vcID string) vcTypes.VerifiableCredential {

	client := vcTypes.NewQueryClient(cc.cos.Context)
	res, err := client.VerifiableCredential(
		context.Background(),
		&vcTypes.QueryVerifiableCredentialRequest{VerifiableCredentialId: vcID},
	)
	if err != nil {
		log.Fatalln("error requesting balance", err)
	}
	return res.GetVerifiableCredential()
}

func (cc *ChainClient) ResolveDID(did string) (didTypes.DidDocument, bool) {

	client := didTypes.NewQueryClient(cc.cos.Context)
	res, err := client.DidDocument(
		context.Background(),
		&didTypes.QueryDidDocumentRequest{Id: did},
	)
	if err != nil {
		return didTypes.DidDocument{}, false
	}
	return res.GetDidDocument(), true
}


