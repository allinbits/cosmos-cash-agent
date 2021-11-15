package ssi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/allinbits/cosmos-cash-agent/pkg/config"

	"github.com/hyperledger/aries-framework-go/component/storageutil/mem"
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

var (
	w      *wallet.Wallet
	client = &http.Client{}
	reqURL string
)

type SSIWallet struct {
	cloudAgentURL     string
	w                 *wallet.Wallet
	ctx               *context.Provider
	didExchangeClient *didexchange.Client
	routeClient       *mediator.Client
	messagingClient   *messaging.Client
}

func createDIDExchangeClient(ctx *context.Provider) *didexchange.Client {
	// create a new did exchange client
	didExchange, err := didexchange.New(ctx)
	if err != nil {
		panic(err)
	}

	actions := make(chan service.DIDCommAction, 1)

	err = didExchange.RegisterActionEvent(actions)
	if err != nil {
		panic(err)
	}

	go func() {
		service.AutoExecuteActionEvent(actions)
	}()

	return didExchange
}

func createRoutingClient(ctx *context.Provider) *mediator.Client {
	// create the mediator client this client handler routing between edge and cloud agents
	routeClient, err := mediator.New(ctx)
	if err != nil {
		panic(err)
	}
	events := make(chan service.DIDCommAction)

	err = routeClient.RegisterActionEvent(events)
	if err != nil {
		panic(err)
	}
	go func() {
		service.AutoExecuteActionEvent(events)
	}()

	return routeClient
}

func createMessagingClient(ctx *context.Provider) *messaging.Client {
	n := LocalNotifier{}
	registrar := msghandler.NewRegistrar()

	msgClient, err := messaging.New(ctx, registrar, n)
	if err != nil {
		panic(err)
	}

	return msgClient

}

func Agent(cfg config.EdgeConfigSchema, pass string) *SSIWallet {
	// datastore
	provider := mem.NewProvider()
	stateProvider := mem.NewProvider()

	// ws outbound
	var transports []transport.OutboundTransport
	outboundWs := ws.NewOutbound()
	transports = append(transports, outboundWs)

	// resolver
	httpVDR, err := httpbinding.New(cfg.CosmosDIDResolverURL,
		httpbinding.WithAccept(func(method string) bool { return method == "cosmos" }))
	if err != nil {
		panic(err.Error())
	}

	// create framework
	framework, err := aries.New(
		aries.WithStoreProvider(provider),
		aries.WithProtocolStateStoreProvider(stateProvider),
		aries.WithOutboundTransports(transports...),
		aries.WithTransportReturnRoute("all"),
		aries.WithVDR(httpVDR),
		//	aries.WithVDR(CosmosVDR{}),
	)
	// get the context
	ctx, err := framework.Context()
	if err != nil {
		panic(err)
	}
	// creating wallet profile using local KMS passphrase
	err = wallet.CreateProfile(cfg.ControllerName, ctx, wallet.WithPassphrase(pass))
	if err != nil {
		panic(err)
	}

	// creating vcwallet instance for user with local KMS settings.
	w, err = wallet.New(cfg.ControllerName, ctx)
	if err != nil {
		panic(err)
	}

	didExchangeClient := createDIDExchangeClient(ctx)
	routeClient := createRoutingClient(ctx)
	messagingClient := createMessagingClient(ctx)

	return &SSIWallet{
		cloudAgentURL:     cfg.CloudAgentPublicURL,
		w:                 w,
		ctx:               ctx,
		didExchangeClient: didExchangeClient,
		routeClient:       routeClient,
		messagingClient:   messagingClient,
	}
}

func (cw *SSIWallet) HandleInvitation(
	invitation *didexchange.Invitation,
) *didexchange.Connection {
	connectionID, err := cw.didExchangeClient.HandleInvitation(invitation)
	if err != nil {
		panic(err)
	}

	connection, err := cw.didExchangeClient.GetConnection(connectionID)
	if err != nil {
		panic(err)
	}
	log.Infoln("Connection created", connection)

	return connection

}

func (cw *SSIWallet) AddMediator(
	connectionID string,
) {
	err := cw.routeClient.Register(connectionID)
	if err != nil {
		panic(err)
	}

	log.Infoln("Mediator created")
}

// Run should be called as a goroutine, the parameters are:
// State: the local state of the app that should be stored on disk
// Hub: is the messages where the 3 components (ui, wallet, agent) can exchange messages
func (cw *SSIWallet) Run(hub *config.MsgHub) {

	// send updates about verifiable credentials
	t0 := time.NewTicker(30 * time.Second)
	go func() {
		for {
			log.Infoln("ticker! retrieving verifiable credentials")
			vcs := []string{}
			hub.Notification <- config.NewAppMsg(config.MsgVCs, vcs)
			<-t0.C
		}
	}()

	// send updates about contacts
	t1 := time.NewTicker(10 * time.Second)
	go func() {
		for {
			// TODO handle contacts
			connections, err := cw.didExchangeClient.QueryConnections(&didexchange.QueryConnectionsParams{})
			if err != nil {
				panic(err)
			}
			log.Infoln("queried connections", connections)
			hub.Notification <- config.NewAppMsg(config.MsgUpdateContacts, connections)
			<-t1.C
		}
	}()

	for {
		m := <-hub.AgentWalletIn
		log.Debugln("received message", m)
		switch m.Typ {
		case config.MsgVCData:
			vcID := m.Payload.(string)
			// TODO: retrieve the verifiable credential
			// vc := cc.GetPublicVC(m.Payload.(string))
			log.Debugln("AgentWallet received MsgVCData msg for ", vcID)
			vc := struct{}{} // <-- fake credential
			// always send to the notification channel for the UI
			// handle the notification in the ui/handlers.go dispatcher function
			hub.Notification <- config.NewAppMsg(m.Typ, vc)
		case config.MsgCreateInvitation:
			log.Debugln(
				"AgentWallet received MsgHandleInvitation msg for ",
				m.Payload.(string),
			)
			// TODO: validate invitation is correct
			var inv didexchange.Invitation
			var jsonStr string
			if m.Payload.(string) != "" {
				inv, err := cw.didExchangeClient.CreateInvitation(
					"bob-alice-connection-1",
					didexchange.WithRouterConnectionID(m.Payload.(string)),
				)
				if err != nil {
					panic(err)
				}
				jsonStr, _ := json.Marshal(inv)
				fmt.Println(string(jsonStr))
			} else {
				inv, err := cw.didExchangeClient.CreateInvitation(
					"bob-alice-conn-direct",
				)
				if err != nil {
					panic(err)
				}
				jsonStr, _ := json.Marshal(inv)
				fmt.Println(string(jsonStr))
			}
			fmt.Print(inv)

			hub.Notification <- config.NewAppMsg(config.MsgUpdateContact, string(jsonStr))
		case config.MsgHandleInvitation:
			log.Debugln(
				"AgentWallet received MsgHandleInvitation msg for ",
				m.Payload.(string),
			)
			var invite de.CreateInvitationResponse

			err := json.Unmarshal([]byte(m.Payload.(string)), &invite.Invitation)
			if err != nil {
				println(err)
			}

			if invite.Invitation == nil {
				reqURL = fmt.Sprint(
					"http://localhost:8090",
					"/connections/create-invitation?&label=BobMediatorEdgeAgent",
				)
				post(client, reqURL, nil, &invite)
			}

			// TODO: validate invitation is correct
			connection := cw.HandleInvitation(invite.Invitation)

			hub.Notification <- config.NewAppMsg(config.MsgContactAdded, connection)
		case config.MsgAddMediator:
			log.Debugln(
				"AgentWallet received MsgHandleInvitation msg for ",
				m.Payload.(string),
			)

			// TODO: validate invitation is correct
			cw.AddMediator(m.Payload.(string))

		case config.MsgGetConnectionStatus:
			connID := m.Payload.(string)
			log.Debugln(
				"AgentWallet received MsgHandleInvitation msg for ",
				connID,
			)
			var sb strings.Builder

			// TODO: validate invitation is correct
			connection, err := cw.didExchangeClient.GetConnection(connID)
			if err != nil {
				panic(err)
			}
			sb.WriteString("ConnectionID: " + connection.ConnectionID + "\n")
			sb.WriteString("Status: " + connection.State + "\n")
			sb.WriteString("Label: " + connection.TheirLabel + "\n")
			routerConfig, err := cw.routeClient.GetConfig(connID)
			log.Info(routerConfig.Endpoint())
			log.Info(routerConfig.Keys())
			if routerConfig != nil {
				sb.WriteString("Mediator: This connection is a mediator" + "\n")
				sb.WriteString("Endpoint: " + routerConfig.Endpoint() + "\n")
				sb.WriteString("Keys: " + strings.Join(routerConfig.Keys(), " ") + "\n")
			}

			hub.Notification <- config.NewAppMsg(config.MsgUpdateContact, sb.String())
		}
	}
}

// TODO remove in favor of public did exchange, here for test purposes
func request(client *http.Client, method, url string, requestBody io.Reader, val interface{}) {
	req, err := http.NewRequest(method, url, requestBody)
	if err != nil {
		fmt.Print(err.Error())
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Print(err.Error())
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Print(err.Error())
	}
	json.Unmarshal(bodyBytes, &val)
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
		panic(err.Error())
	}
	return bytes.NewBuffer(v)
}

// AcceptContactRequest
// SendContactRequest
// AcceptVC
// RequestVC
