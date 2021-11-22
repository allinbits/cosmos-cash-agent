package chain

import (
	"context"
	"github.com/allinbits/cosmos-cash-agent/pkg/config"
	"github.com/tendermint/starport/starport/pkg/cosmosclient"
	"strings"
	"time"

	didTypes "github.com/allinbits/cosmos-cash/v2/x/did/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	log "github.com/sirupsen/logrus"
)

type ChainClient struct {
	cos cosmosclient.Client
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

	api := strings.Replace(cfg.NodeURI, "https://rpc.", "https://api.", 1)
	log.Infoln("api address", api)

	cosmos, err := cosmosclient.New(context.Background(),
		cosmosclient.WithHome(chainData),
		cosmosclient.WithNodeAddress(cfg.NodeURI),
		cosmosclient.WithUseFaucet(cfg.FaucetURL, "cash", 10),
		cosmosclient.WithAPIAddress(api),
	)

	if err != nil {
		log.Fatalln("chain client init error", err)
	}

	_, err = cosmos.Account(cfg.ControllerName)
	if err != nil {
		i, m, _ := cosmos.AccountRegistry.Create(cfg.ControllerName)
		log.WithFields(log.Fields{"mnemonic": m, "address": i.Info.GetAddress()}).Info("new account created")
	}
	log.Infoln("opening existing client for", cfg.ControllerName)
	// open account
	a, err := cosmos.Account(cfg.ControllerName)
	if err != nil {
		log.Fatalln("opening account error", err)
	}

	cc := ChainClient{
		cos: cosmos,
		acc: a.Info.GetAddress(),
		did: didTypes.NewChainDID(cfg.ChainID, cfg.ControllerDidID),
	}

	if _, ok := cc.ResolveDID(didTypes.NewChainDID(cc.cos.Context.ChainID, cfg.ControllerDidID).String()); !ok {
		msg := initDIDDoc(cfg.ChainID, cfg.ControllerDidID, cfg.CloudAgentPublicURL, a.Info)
		_, err = cc.cos.BroadcastTx(cfg.ControllerName, msg)
		if err != nil {
			log.Fatalln("cannot broadcast here", err, "|||",  a.Info.GetAddress().String())
		}
	}
	return &cc
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

	//// add verification
	//return didTypes.NewMsgAddVerification(
	//	did.String(),
	//	verification,
	//	ki.GetAddress().String(),
	//)
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
		switch m.Typ {
		case config.MsgPublicVCData:
			vc := cc.GetPublicVC(m.Payload.(string))
			log.Debugln("TokenWallet received MsgPublicVCData msg for ", m.Payload.(string))
			hub.Notification <- config.NewAppMsg(m.Typ, vc)
		case config.MsgChainOfTrust:
			coinStr := m.Payload.(string)
			c, _ := sdk.ParseCoinNormalized(coinStr)
			cot := cc.GetDenomChainOfTrust(c.GetDenom())
			hub.Notification <- config.NewAppMsg(m.Typ, cot)
		case config.MsgMarketplaceData:
			licenseCredentialID := m.Payload.(string)
			cot := cc.GetChainOfTrust(licenseCredentialID)
			hub.Notification <- config.NewAppMsg(m.Typ, cot)
		}
	}
}
