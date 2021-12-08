package ssi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/allinbits/cosmos-cash-agent/pkg/helpers"
	"github.com/allinbits/cosmos-cash-agent/pkg/model"
	"github.com/hyperledger/aries-framework-go/component/storage/leveldb"
	"github.com/hyperledger/aries-framework-go/pkg/doc/verifiable"
	"github.com/hyperledger/aries-framework-go/pkg/kms"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/allinbits/cosmos-cash-agent/pkg/config"

	"github.com/hyperledger/aries-framework-go/pkg/client/didexchange"
	"github.com/hyperledger/aries-framework-go/pkg/client/mediator"
	"github.com/hyperledger/aries-framework-go/pkg/client/messaging"

	de "github.com/hyperledger/aries-framework-go/pkg/controller/command/didexchange"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/common/service"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/messaging/msghandler"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/transport"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/transport/ws"
	"github.com/hyperledger/aries-framework-go/pkg/framework/aries"
	"github.com/hyperledger/aries-framework-go/pkg/framework/context"
	"github.com/hyperledger/aries-framework-go/pkg/vdr/httpbinding"
	"github.com/hyperledger/aries-framework-go/pkg/wallet"

	log "github.com/sirupsen/logrus"
)

type genericChatMsg struct {
	ID           string   `json:"@id"`
	Type         string   `json:"@type"`
	Purpose      []string `json:"~purpose"`
	Message      string   `json:"message"`
	From         string   `json:"from"`
	SenderDID    string   `json:"senderDID"`
	RecipientDID string   `json:"recipientDID"`
}

var (
	w      *wallet.Wallet
	client = &http.Client{}
	reqURL string
)

// SSIWallet is the wallet
type SSIWallet struct {
	cloudAgentURL     string
	cloudAgentAPI     string
	cloudAgentWsURL   string
	ControllerDID     string
	ControllerName    string
	MediatorDID       string
	w                 *wallet.Wallet
	ctx               *context.Provider
	didExchangeClient *didexchange.Client
	routeClient       *mediator.Client
	messagingClient   *messaging.Client
	walletAuthToken   string
}

func (s SSIWallet) GetContext() *context.Provider {
	return s.ctx
}

func (s SSIWallet) GetMessagingClient() *messaging.Client {
	return s.messagingClient
}

func createDIDExchangeClient(ctx *context.Provider) *didexchange.Client {
	// create a new did exchange client
	didExchange, err := didexchange.New(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	actions := make(chan service.DIDCommAction, 1)

	err = didExchange.RegisterActionEvent(actions)
	if err != nil {
		log.Fatalln(err)
	}

	// NOTE: no auto execute because it doens't work with routing
	//	go func() {
	//		service.AutoExecuteActionEvent(actions)
	//	}()

	return didExchange
}

func createRoutingClient(ctx *context.Provider) *mediator.Client {
	// create the mediator client this client handler routing between edge and cloud agents
	routeClient, err := mediator.New(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	events := make(chan service.DIDCommAction)

	err = routeClient.RegisterActionEvent(events)
	if err != nil {
		log.Fatalln(err)
	}
	go func() {
		service.AutoExecuteActionEvent(events)
	}()

	return routeClient
}

func createMessagingClient(
	ctx *context.Provider,
	registrar *msghandler.Registrar,
	runtimeMsgs *config.MsgHub,
) *messaging.Client {
	n := NewNotifier(ctx, runtimeMsgs)

	msgClient, err := messaging.New(ctx, registrar, n)
	if err != nil {
		log.Fatalln(err)
	}
	msgType := "https://didcomm.org/generic/1.0/message"
	purpose := []string{"meeting", "appointment", "event"}
	name := "generic-message"

	err = msgClient.RegisterService(name, msgType, purpose...)
	if err != nil {
		log.Fatalln(err)
	}

	return msgClient
}

func Agent(cfg config.EdgeConfigSchema, pass string) *SSIWallet {
	// datastore
	storePath, _ := config.GetAppData("aries_store")
	storeProvider := leveldb.NewProvider(storePath)

	statePath, _ := config.GetAppData("aries_state")
	stateProvider := leveldb.NewProvider(statePath)

	// ws outbound
	var transports []transport.OutboundTransport
	outboundWs := ws.NewOutbound()
	transports = append(transports, outboundWs)

	// resolver
	httpVDR, err := httpbinding.New(cfg.CosmosDIDResolverURL,
		httpbinding.WithAccept(func(method string) bool { return method == "cosmos" }))
	if err != nil {
		log.Fatalln(err)
	}
	registrar := msghandler.NewRegistrar()

	// create framework
	framework, err := aries.New(
		aries.WithStoreProvider(storeProvider),
		aries.WithProtocolStateStoreProvider(stateProvider),
		aries.WithOutboundTransports(transports...),
		aries.WithMessageServiceProvider(registrar),
		aries.WithTransportReturnRoute("all"),
		aries.WithKeyType(kms.ED25519Type),
		aries.WithKeyAgreementType(kms.X25519ECDHKWType),
		aries.WithVDR(httpVDR),
		//	aries.WithVDR(CosmosVDR{}),
	)

	// get the context
	ctx, err := framework.Context()
	if err != nil {
		log.Fatalln(err)
	}

	didExchangeClient := createDIDExchangeClient(ctx)
	routeClient := createRoutingClient(ctx)
	messagingClient := createMessagingClient(ctx, registrar, cfg.RuntimeMsgs)

	genKeys := false
	// creating wallet profile using local KMS passphrase
	if err := wallet.CreateProfile(cfg.ControllerName, ctx, wallet.WithPassphrase(pass)); err != nil {
		log.Infoln("profile already exists for", cfg.ControllerName, err)
	} else {
		genKeys = true
	}

	// creating vcwallet instance for user with local KMS settings.
	log.Infoln("opening wallet for", cfg.ControllerName)
	w, err = wallet.New(cfg.ControllerName, ctx)
	if err != nil {
		log.Fatalln(err)
	}

	// TODO the wallet should be closed eventually
	walletAuthToken, err := w.Open(wallet.WithUnlockByPassphrase(pass))
	if err != nil {
		log.Fatalln("wallet cannot be opened", err)
	} else {
		log.Println("wallet auth token is", walletAuthToken)
	}

	if genKeys {
		log.Infoln("creating keys for", cfg.ControllerName)
		// create a key to perform keyAgreement between agent
		keyID, pubKeyBytes, err := ctx.KMS().CreateAndExportPubKeyBytes(kms.X25519ECDHKWType)
		if err != nil {
			log.Fatalln("cannot create X25519ECDHKWType key")
		}
		kp := &wallet.KeyPair{
			KeyID:     keyID,
			PublicKey: base64.RawURLEncoding.EncodeToString(pubKeyBytes),
		}
		// send data to the token wallet
		cfg.RuntimeMsgs.TokenWalletIn <- config.NewAppMsg(config.MsgDIDAddVerificationMethod, model.X25519{KeyPair: kp})

		// create a key to sign credentials
		wkp, err := w.CreateKeyPair(walletAuthToken, kms.ED25519Type)
		if err != nil {
			log.Fatalln("cannot create ED25519Type key")
		}
		cfg.RuntimeMsgs.TokenWalletIn <- config.NewAppMsg(config.MsgDIDAddVerificationMethod, model.ED25519{KeyPair: wkp})
	}

	return &SSIWallet{
		cloudAgentURL:     cfg.CloudAgentPublicURL,
		ControllerDID:     cfg.ControllerDID(),
		ControllerName:    cfg.ControllerName,
		cloudAgentWsURL:   cfg.CloudAgentWsURL,
		cloudAgentAPI:     cfg.CloudAgentAPIURL(),
		MediatorDID:       cfg.MediatorDID(),
		w:                 w,
		ctx:               ctx,
		didExchangeClient: didExchangeClient,
		routeClient:       routeClient,
		messagingClient:   messagingClient,
		walletAuthToken:   walletAuthToken,
	}
}

func (s *SSIWallet) HandleInvitation(
	invitation *didexchange.Invitation,
) (connection *didexchange.Connection) {
	connectionID, err := s.didExchangeClient.HandleInvitation(invitation)
	if err != nil {
		log.Errorln(err)
		return
	}

	connection, err = s.didExchangeClient.GetConnection(connectionID)
	if err != nil {
		log.Errorln(err)
		return
	}
	log.WithFields(log.Fields{"connectionID": connectionID}).Infoln("Connection created", connection)
	return connection
}

func (s *SSIWallet) AddMediator(
	connectionID string,
) {
	err := s.routeClient.Register(connectionID)
	if err != nil {
		log.Fatalln(err)
	}

	log.Infoln("Mediator created")
}

func (s *SSIWallet) Close() {
	s.w.Close()
}

// Run should be called as a goroutine, the parameters are:
// State: the local state of the app that should be stored on disk
// Hub: is the messages where the 3 components (ui, wallet, agent) can exchange messages
func (s *SSIWallet) Run(hub *config.MsgHub) {

	// send updates about verifiable credentials
	t0 := time.NewTicker(30 * time.Second)
	go func() {
		for {
			log.Infoln("ticker! retrieving verifiable credentials")
			var vcs []verifiable.Credential
			if credentials, err := s.w.GetAll(s.walletAuthToken, wallet.Credential); err == nil {
				for _, vcRaw := range credentials {
					b, _ := vcRaw.MarshalJSON()
					var vc verifiable.Credential
					json.Unmarshal(b, &vc)
					vcs = append(vcs, vc)
				}
			} else {
				log.Errorln("failed to read credentials from wallet", err)
			}
			hub.Notification <- config.NewAppMsg(config.MsgVCs, vcs)
			<-t0.C
		}
	}()

	// send updates about contacts
	t1 := time.NewTicker(10 * time.Second)
	go func() {
		for {
			connections, err := s.didExchangeClient.QueryConnections(&didexchange.QueryConnectionsParams{})
			if err != nil {
				log.Fatalln(err)
			}
			// retrieve the connections
			savedConnections, _ := s.w.GetAll(s.walletAuthToken, wallet.Connection)
			connectionContacts := make(map[string]model.Contact, len(savedConnections))
			for _, jrmc := range savedConnections {
				var c didexchange.Connection
				json.Unmarshal(jrmc, &c)
				connectionContacts[c.ConnectionID] = model.NewContact(&c)
			}
			// now search if the connection exists
			for _, c := range connections {
				if _, found := connectionContacts[c.ConnectionID]; !found {
					s.w.Add(s.walletAuthToken, wallet.Connection, helpers.RawJson(c))
					connectionContacts[c.ConnectionID] = model.NewContact(c)
				}
			}
			// sort the contacts
			var contacts []interface{}
			keys := make([]string, 0, len(connectionContacts))
			for k := range connectionContacts {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				contacts = append(contacts, connectionContacts[k])
			}
			// send them to the ui
			hub.Notification <- config.NewAppMsg(config.MsgUpdateContacts, contacts)
			<-t1.C
		}
	}()

	for {
		m := <-hub.AgentWalletIn
		log.Debugln("received message", m)
		switch m.Typ {
		case config.MsgIssueVC:
			// https://github.com/hyperledger/aries-framework-go/blob/main/docs/vc_wallet.md#add
			ce := m.Payload.(model.CallableEnvelope)
			log.WithFields(log.Fields{"credential": ce.DataIn}).Debugln("adding credential")
			// issue the credential
			signedVC, err := s.w.Issue(s.walletAuthToken, ce.DataIn.(json.RawMessage), &wallet.ProofOptions{
				Controller: s.ControllerDID,
			})
			if err != nil {
				log.Errorln("error issuing credential", err)
				break
			}
			// now convert the vc to string
			rawSignedVC, _ := signedVC.MarshalJSON()
			log.Infof("issued credential %s", rawSignedVC)
			// now trigger the function in the envelope
			ce.Callback(string(rawSignedVC))
		case config.MsgSSIAddVC:
			vcStr := m.Payload.(string)
			if err := s.w.Add(s.walletAuthToken, wallet.Credential, json.RawMessage(vcStr)); err != nil {
				log.Errorln("error adding credential to the wallet", err)
				break
			}
			log.Debugln("private credential added to the wallet")
		case config.MsgVCData:
			vcID := m.Payload.(string)
			vc, err := s.w.Get(s.walletAuthToken, wallet.Credential, vcID)
			if err != nil {
				log.Errorln("cannot retrieve the credential ", vcID, err)
			}
			// always send to the notification channel for the UI
			// handle the notification in the ui/handlers.go dispatcher function
			hub.Notification <- config.NewAppMsg(m.Typ, vc)
		case config.MsgCreateInvitation:
			log.Debugln(
				"AgentWallet received MsgHandleInvitation msg for ",
				m.Payload.(string),
			)
			var inv didexchange.Invitation
			var jsonStr string
			if m.Payload.(string) != "" {
				// TODO: is it correct here the controller name ? seems so
				inv, err := s.didExchangeClient.CreateInvitation(
					s.ControllerName,
					didexchange.WithRouterConnectionID(m.Payload.(string)),
				)
				if err != nil {
					log.Errorln(err)
				}
				jsonStr, _ := json.Marshal(inv)
				log.Debugln("create invitation reply", string(jsonStr))
				// copy the invitation to clipboard
				hub.Notification <- config.NewAppMsg(config.MsgClipboard, string(jsonStr))
				fmt.Println(string(jsonStr))
			} else {
				// TODO: is it correct here the controller name ? seems so
				inv, err := s.didExchangeClient.CreateInvitation(
					s.ControllerName,
				)
				if err != nil {
					log.Fatalln(err)
				}
				jsonStr, _ := json.Marshal(inv)
				log.Debugln("direct create invitation", string(jsonStr))
			}
			log.Debugln("invitation is", inv)

			hub.Notification <- config.NewAppMsg(config.MsgUpdateContact, string(jsonStr))
		case config.MsgHandleInvitation:
			log.Debugln(
				"AgentWallet received MsgHandleInvitation msg for ",
				m.Payload.(string),
			)
			var invite de.CreateInvitationResponse

			if err := json.Unmarshal([]byte(m.Payload.(string)), &invite.Invitation); err != nil {
				log.Errorln("error unmarshalling the invitation in HsgHandleInvitation, requesting a new one")

				reqURL = fmt.Sprint(
					s.cloudAgentAPI,
					fmt.Sprintf("/connections/create-invitation?public=%s&label=TDMMediatorEdgeAgent", s.MediatorDID),
				)
				post(client, reqURL, nil, &invite)
			}
			log.Infoln("invitation is ", invite)
			// TODO: validate invitation is correct
			connection := s.HandleInvitation(invite.Invitation)
			s.w.Add(s.walletAuthToken, wallet.Connection, helpers.RawJson(connection))
			// hub.Notification <- config.NewAppMsg(config.MsgContactAdded, connection)
		case config.MsgApproveInvitation:
			log.Debugln(
				"AgentWallet received MsgHandleInvitation msg for ",
				m.Payload.(string),
			)
			params := strings.Split(m.Payload.(string), " ")

			if len(params) > 1 && params[1] != "" {
				err := s.didExchangeClient.AcceptInvitation(
					params[0],
					"",
					s.ControllerName,
					didexchange.WithRouterConnections(params[1]))
				if err != nil {
					log.Errorln("AcceptInvitation", err)
					break
				}
			} else {
				err := s.didExchangeClient.AcceptInvitation(
					params[0],
					"",
					s.ControllerName,
				)
				if err != nil {
					log.Errorln("AcceptInvitation", err)
					break
				}
			}
		case config.MsgApproveRequest:
			log.Debugln(
				"AgentWallet received MsgHandleInvitation msg for ",
				m.Payload.(string),
			)
			params := strings.Split(m.Payload.(string), " ")
			err := s.didExchangeClient.AcceptExchangeRequest(
				params[0],
				"",
				s.ControllerName,
				didexchange.WithRouterConnections(params[1]),
			)
			if err != nil {
				log.Errorln("AcceptExchangeRequest", err)
				break
			}

		case config.MsgAddMediator:
			log.Debugln(
				"AgentWallet received MsgHandleInvitation msg for ",
				m.Payload.(string),
			)

			// TODO: validate invitation is correct
			s.AddMediator(m.Payload.(string))

		case config.MsgGetConnectionStatus:
			connID := m.Payload.(string)
			log.Debugln(
				"AgentWallet received MsgHandleInvitation msg for ",
				connID,
			)
			var sb strings.Builder

			// TODO: validate invitation is correct
			connection, err := s.didExchangeClient.GetConnection(connID)
			if err != nil {
				log.Errorln("GetConnection", err)
				break
			}
			sb.WriteString("ConnectionID: " + connection.ConnectionID + "\n")
			sb.WriteString("Status: " + connection.State + "\n")
			sb.WriteString("Label: " + connection.TheirLabel + "\n")
			routerConfig, err := s.routeClient.GetConfig(connID)
			if routerConfig != nil {
				log.Info(routerConfig.Endpoint())
				log.Info(routerConfig.Keys())
				sb.WriteString("Mediator: This connection is a mediator" + "\n")
				sb.WriteString("Endpoint: " + routerConfig.Endpoint() + "\n")
				sb.WriteString("Keys: " + strings.Join(routerConfig.Keys(), " ") + "\n")
			}

			hub.Notification <- config.NewAppMsg(config.MsgUpdateContact, sb.String())
		case config.MsgSendText:

			tm := m.Payload.(model.TextMessage)
			log.Debugln(
				"AgentWallet received MsgSendText msg for ",
				tm.Content,
			)

			connection, err := s.didExchangeClient.GetConnection(tm.Channel)
			if err != nil {
				log.Errorln("GetConnection", err)
				break
			}

			var genericMsg genericChatMsg
			genericMsg.ID = fmt.Sprint(rand.Float64())
			genericMsg.Type = "https://didcomm.org/generic/1.0/message"
			genericMsg.Purpose = []string{"meeting", "appointment", "event"}
			genericMsg.Message = tm.Content
			genericMsg.From = s.ControllerName
			genericMsg.SenderDID = connection.MyDID
			genericMsg.RecipientDID = connection.TheirDID

			rawBytes, _ := json.Marshal(genericMsg)
			resp, err := s.messagingClient.Send(rawBytes, messaging.SendByConnectionID(tm.Channel))
			if err != nil {
				log.Errorln("SendByConnectionID", err)
			}
			log.Debugln("message response is", resp)
		case config.MsgDeleteConnection:
			targetConnectionID := m.Payload.(string)
			// find and delete the connection
			savedConnections, _ := s.w.GetAll(s.walletAuthToken, wallet.Connection)
			for k, rc := range savedConnections {
				var c didexchange.Connection
				_ = json.Unmarshal(rc, &c)
				log.Debugf("checking connection %s for removal against %s", c.ConnectionID, targetConnectionID)
				if c.ConnectionID == targetConnectionID {
					log.Debugln("removing connection ", c.ConnectionID, k)
					s.w.Remove(s.walletAuthToken, wallet.Connection, k)
					s.didExchangeClient.RemoveConnection(c.ConnectionID)
				}
			}
		}
	}
}

// TODO remove in favor of public did exchange, here for test purposes
func request(client *http.Client, method, url string, requestBody io.Reader, val interface{}) {
	req, err := http.NewRequest(method, url, requestBody)
	log.Debugln("executing http request", req)
	if err != nil {
		log.Errorln(err)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Errorln(err)
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorln(err)
	}
	if err := json.Unmarshal(bodyBytes, &val); err != nil {
		log.Errorln(err)
	}
}

func post(client *http.Client, url string, requestBody, val interface{}) {
	if requestBody != nil {
		request(client, "POST", url, bitify(requestBody), val)
	} else {
		request(client, "POST", url, nil, val)
	}

}
func bitify(in interface{}) io.Reader {
	v, err := json.Marshal(in)
	if err != nil {
		log.Errorln(err)
	}
	return bytes.NewBuffer(v)
}

// AcceptVC
// RequestVC
