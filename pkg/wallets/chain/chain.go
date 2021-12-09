package chain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/allinbits/cosmos-cash-agent/pkg/config"
	"github.com/allinbits/cosmos-cash-agent/pkg/helpers"
	"github.com/allinbits/cosmos-cash-agent/pkg/model"
	issuerTypes "github.com/allinbits/cosmos-cash/v2/x/issuer/types"
	regulatorTypes "github.com/allinbits/cosmos-cash/v2/x/regulator/types"
	vcTypes "github.com/allinbits/cosmos-cash/v2/x/verifiable-credential/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"google.golang.org/grpc"
	"net/http"
	"time"

	"github.com/allinbits/cosmos-cash/v2/app"
	didTypes "github.com/allinbits/cosmos-cash/v2/x/did/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

type ChainClient struct {
	ctx client.Context
	fct tx.Factory
	qry grpc.ClientConn
	acc sdk.AccAddress
	did didTypes.DID
}

type KeyData struct {
	Address string `json:"address"`
	Armor   string `json:"armor"`
}

func Client(cfg config.EdgeConfigSchema, password string) *ChainClient {
	log.Infoln("initializing client")
	chainData, _ := config.GetAppData("chain")

	kr := keyring.NewInMemory()

	armorPath, exists := config.GetAppData("account_armor.json")
	if !exists {
		info, mnemonic, err := kr.NewMnemonic(cfg.ControllerName, keyring.English, sdk.GetConfig().GetFullBIP44Path(), "", hd.Secp256k1)
		if err != nil {
			log.Fatalln("error creating new key", err)
		}
		log.WithFields(log.Fields{
			"mnemonic": mnemonic,
		}).Infoln("created new key for", cfg.ControllerName)
		armorData, err := kr.ExportPrivKeyArmor(cfg.ControllerName, password)
		if err != nil {
			log.Fatalln("error exporting private key", err)
		}

		helpers.WriteJson(armorPath, KeyData{
			Address: info.GetAddress().String(),
			Armor:   armorData,
		})
		log.Infoln("exported armored private key to", armorPath)
	} else {
		var accountData KeyData
		helpers.LoadJson(armorPath, &accountData)
		err := kr.ImportPrivKey(cfg.ControllerName, accountData.Armor, password)
		if err != nil {
			log.Fatalln("error loading private key", err)
		}
		log.Infoln("private key loaded from", armorPath)
	}
	// now get the account
	ki, err := kr.Key(cfg.ControllerName)
	if err != nil {
		log.Fatalln("cannot load stored key by uid", err)
	}
	// RPC client for transactions
	netCli, err := client.NewClientFromNode(cfg.NodeURI)
	if err != nil {
		log.Fatalln("error connecting to the node", err)
	}
	encodingConfig := app.MakeEncodingConfig()
	initClientCtx := client.Context{}.
		WithCodec(encodingConfig.Marshaler).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithAccountRetriever(authTypes.AccountRetriever{}).
		WithBroadcastMode(flags.BroadcastBlock).
		WithChainID(cfg.ChainID).
		WithKeyring(kr).
		WithHomeDir(chainData).
		WithNodeURI(cfg.NodeURI).
		WithFromName(cfg.ControllerName).
		WithFromAddress(ki.GetAddress()).
		WithSkipConfirmation(true).
		WithClient(netCli)
	//WithLegacyAmino(encodingConfig.Amino).
	//WithInput(os.Stdin).

	log.Infoln("opening existing client for", cfg.ControllerName)
	pf := pflag.NewFlagSet("default", pflag.PanicOnError)
	factory := tx.NewFactoryCLI(initClientCtx, pf).
		WithChainID(initClientCtx.ChainID).
		WithAccountRetriever(initClientCtx.AccountRetriever).
		WithKeybase(initClientCtx.Keyring)

	cc := ChainClient{
		ctx: initClientCtx,
		fct: factory,
		acc: ki.GetAddress(),
		did: didTypes.NewChainDID(cfg.ChainID, cfg.ControllerDidID),
	}

	if !exists {
		// get some tokens here
		if !cc.GetBalance(ki.GetAddress().String()).IsPositive() {
			callFaucet(cfg.FaucetURL, ki.GetAddress().String())

			for i := 0; i < 3; i++ {
				time.Sleep(6 * time.Second)
				if cc.GetBalance(ki.GetAddress().String()).IsPositive() {
					log.Infoln("got a positive balance")
					break
				}
			}
		}
		msg := initDIDDoc(
			cfg.ChainID,
			cfg.ControllerDidID,
			cfg.CloudAgentPublicURL,
			ki,
		)
		cc.BroadcastTx(msg)
	}

	return &cc
}

// BroadcastTx broadcast the transaction and retrieve the tx hash
func (cc *ChainClient) BroadcastTx(msgs ...sdk.Msg) (txHash string) {
	log.Infoln("broadcasting messages")
	w := cc.ctx.Output
	// set the ctx output to a buffer
	b := new(bytes.Buffer)
	cc.ctx.Output = b
	// execute the tx
	if err := tx.GenerateOrBroadcastTxWithFactory(cc.ctx, cc.fct, msgs...); err != nil {
		log.Fatalln("failed tx", err)
	}
	// restore the buffer
	cc.ctx.Output = w
	// parse the json and retrieve the tx hash
	tx := make(map[string]interface{})
	if err := json.Unmarshal(b.Bytes(), &tx); err != nil {
		log.Errorln("error unmarshalling transaction ", err)
		return
	}
	txHash = tx["txhash"].(string)
	return
}

func (cc *ChainClient) Close() {
}

// Init performs the client initialization
// 1. creates account keypair
// 2. get some coins from the faucet
// 3. creates the did document
func initDIDDoc(chainID, didID, agentURL string, ki keyring.Info) sdk.Msg {
	log.Println("initializing new did document", didID)
	did := didTypes.NewChainDID(chainID, didID)
	// verification method id
	vmID := did.NewVerificationMethodID(ki.GetAddress().String())

	verification := didTypes.NewVerification(
		didTypes.NewVerificationMethod(
			vmID,
			did,
			didTypes.NewPublicKeyMultibase(ki.GetPubKey().Bytes(), didTypes.DIDVMethodTypeEcdsaSecp256k1VerificationKey2019),
		),
		[]string{didTypes.Authentication},
		nil,
	)

	service := didTypes.NewService(
		didID+"-agent",
		"DIDCommMessaging",
		agentURL,
	)

	return didTypes.NewMsgCreateDidDocument(
		did.String(),
		didTypes.Verifications{verification},
		didTypes.Services{service},
		ki.GetAddress().String(),
	)
}

func callFaucet(faucetURL, address string) {
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}
	payload := fmt.Sprintf(`{"address": "%s"}`, address)
	rsp, err := netClient.Post(faucetURL, "application/json", bytes.NewReader([]byte(payload)))
	if err != nil {
		log.Fatalln("error requesting tokens from the faucet")
	}
	log.Debugf("faucet response: %v", rsp)
	if rsp.StatusCode != http.StatusOK {
		log.Fatalln("error requesting tokens from the faucet", rsp.Status, rsp)
	}

}

func (cc *ChainClient) Run(hub *config.MsgHub) {

	// send updates about balances
	t0 := time.NewTicker(30 * time.Second)
	go func() {
		for {
			log.Infoln("ticker! retrieving balances for ", cc.acc)
			balances := cc.GetBalances(cc.acc.String())
			hub.Notification <- config.NewAppMsg(config.MsgBalances, balances)
			<-t0.C
		}
	}()

	// send updates about credentials
	t1 := time.NewTicker(30 * time.Second)
	go func() {
		for {
			log.Infoln("ticker! retrieving credentials for ", cc.did)
			vcs := cc.GetHolderPublicVCS(cc.did.String())
			hub.Notification <- config.NewAppMsg(config.MsgPublicVCs, vcs)
			<-t1.C
		}
	}()

	// send updates about credentials
	t3 := time.NewTicker(30 * time.Second)
	go func() {
		for {
			log.Infoln("ticker! retrieving marketplaces for ", cc.did)
			vcs := cc.GetLicenseCredentials()
			hub.Notification <- config.NewAppMsg(config.MsgMarketplaces, vcs)
			<-t3.C
		}
	}()

	// now process incoming queue
	for {
		m := <-hub.TokenWalletIn
		log.Debugf("token wallet received message %v", m)
		switch m.Typ {
		case config.MsgPublicVCData:
			vc := cc.GetPublicVC(m.Payload.(string))
			log.Debugln("TokenWallet received MsgPublicVCData msg for ", m.Payload.(string))
			hub.Notification <- config.NewAppMsg(m.Typ, vc)
		case config.MsgChainOfTrust:
			log.Debugln("TokenWallet received MsgPublicVCData msg for ", m.Payload.(string))
			coinStr := m.Payload.(string)
			c, _ := sdk.ParseCoinNormalized(coinStr)
			cot := cc.GetDenomChainOfTrust(c.GetDenom())
			hub.Notification <- config.NewAppMsg(m.Typ, cot)
		case config.MsgMarketplaceData:
			licenseCredentialID := m.Payload.(string)
			cot := cc.GetChainOfTrust(licenseCredentialID)
			hub.Notification <- config.NewAppMsg(m.Typ, cot)
		case config.MsgDIDAddVerificationMethod:
			log.Debugln("adding new verification  method to the did", cc.did)
			apk := m.Payload.(model.AriesPubKey)
			cc.DIDAddVerification(apk.KeyID(), apk.PubKeyBytes(), apk.VerificationMaterialType(), apk.DIDRelationships()...)
		case config.MsgChainAddAddress:
			// TODO GENERATE A NEW ACCOUNT ADDRESS
			hub.AgentWalletIn <- config.NewAppMsg(config.MsgIssueVC, model.NewCallableEnvelope(
				helpers.RawJson(model.ChainAccountCredential(cc.ctx.ChainID, cc.acc.String(), cc.did.String(), fmt.Sprint("Main wallet: ", cc.acc.String()[0:10]))),
				func(signedVC string) {
					hub.AgentWalletIn <- config.NewAppMsg(config.MsgSSIAddVC, signedVC)
				}),
			)
		case config.MsgChainGetAddresses:
			envelope := m.Payload.(model.CallableEnvelope)
			envelope.Callback(cc.acc.String())
		case config.MsgPaymentRequest:
			pr := m.Payload.(model.PaymentRequest)
			// do the payment
			recipient, err := sdk.AccAddressFromBech32(pr.Recipient)
			if err != nil {
				log.Errorln(err)
			}
			amount, err := sdk.ParseCoinNormalized(fmt.Sprintf("%v%s", pr.Amount, pr.Denom))
			if err != nil {
				log.Errorln(err)
			}
			msg := banktypes.NewMsgSend(cc.acc, recipient, sdk.Coins{amount})
			txHash := cc.BroadcastTx(msg)
			hub.AgentWalletIn <- config.NewAppMsg(config.MsgIssueVC, model.NewCallableEnvelope(
				helpers.RawJson(model.NewPaymentReceiptCredential(cc.did.String(), txHash, pr)),
				func(signedVC string) {
					hub.AgentWalletIn <- config.NewAppMsg(config.MsgSSIAddVC, signedVC)
					log.Infoln("payment sent", signedVC)
				}),
			)
		case config.MsgIssueVC:

			vc := m.Payload.(vcTypes.VerifiableCredential)
			signedVC, err := vc.Sign(cc.fct.Keybase(), cc.acc, vc.GetIssuerDID().NewVerificationMethodID(cc.acc.String()))
			if err != nil {
				log.Errorln("error signing public verifiable credential", err)
			}

			var msg sdk.Msg
			switch vc.GetCredentialSubject().(type) {
			case *vcTypes.VerifiableCredential_UserCred:
				msg = issuerTypes.NewMsgIssueUserCredential(signedVC, cc.acc.String())
			case *vcTypes.VerifiableCredential_LicenseCred:
				msg = regulatorTypes.NewMsgIssueLicenseCredential(signedVC, cc.acc.String())
			case *vcTypes.VerifiableCredential_RegistrationCred:
				msg = regulatorTypes.NewMsgIssueRegistrationCredential(signedVC, cc.acc.String())
			case *vcTypes.VerifiableCredential_RegulatorCred:
				msg = regulatorTypes.NewMsgIssueRegulatorCredential(signedVC, cc.acc.String())
			}
			txHash := cc.BroadcastTx(msg)
			log.WithFields(log.Fields{"json": helpers.ToJson(vc)}).Println("transaction hash", txHash)
		}
	}
}
