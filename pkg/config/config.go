package config

import (
	"fmt"
	"github.com/allinbits/cosmos-cash-agent/pkg/helpers"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
)

// NewEdgeConfigSchema sensible defaults for configuration
func NewEdgeConfigSchema(controllerName string) EdgeConfigSchema {
	if helpers.Env("CASH_ENV", "live") == "dev" {
		return EdgeConfigSchema{
			ControllerName:       controllerName,
			NodeURI:              helpers.Env("NODE_URI", "http://127.0.0.1:26657"),
			FaucetURL:            helpers.Env("FAUCET_URL", "https://faucet.cosmos-cash.app.beta.starport.cloud"),
			ChainID:              helpers.Env("CHAIN_ID", "cash"),
			CloudAgentWsURL:      helpers.Env("AGENT_WS_URL", "wss://ws.agent.cosmos-cash.app.beta.starport.cloud"),
			CloudAgentPublicURL:  helpers.Env("AGENT_PUBLIC_URL", "http://localhost:8090"),
			CosmosDIDResolverURL: helpers.Env("RESOLVER_URL", "http://localhost:2109/identifier/aries/"),
			ControllerDidID:      uuid.New().String(),
			// runtime
			RuntimeMsgs: NewMsgHub(),
		}
	}

	return EdgeConfigSchema{
		ControllerName:       controllerName,
		NodeURI:              helpers.Env("NODE_URI", "https://rpc.cosmos-cash.app.beta.starport.cloud:443"),
		FaucetURL:            helpers.Env("FAUCET_URL", "https://faucet.cosmos-cash.app.beta.starport.cloud"),
		ChainID:              helpers.Env("CHAIN_ID", "cosmoscash-testnet"),
		CloudAgentWsURL:      helpers.Env("AGENT_WS_URL", "wss://ws.agent.cosmos-cash.app.beta.starport.cloud"),
		CloudAgentPublicURL:  helpers.Env("AGENT_PUBLIC_URL", "https://agent.cosmos-cash.app.beta.starport.cloud"),
		CosmosDIDResolverURL: helpers.Env("RESOLVER_URL", "https://resolver-driver.cosmos-cash.app.beta.starport.cloud/identifier/aries"),
		ControllerDidID:      uuid.New().String(),
		// runtime
		RuntimeMsgs: NewMsgHub(),
	}
}

type EdgeConfigSchema struct {
	ControllerName       string `json:"controller_name"`
	ControllerDidID      string `json:"controller_did"`
	NodeURI              string `json:"node_uri"`
	FaucetURL            string `json:"faucet_url"`
	ChainID              string `json:"chain_id"`
	CloudAgentWsURL      string `json:"cloud_agent_ws_url"`
	CloudAgentPublicURL  string `json:"cloud_agent_public_url"`
	CosmosDIDResolverURL string `json:"cosmos_did_resolver_url"`

	// Runtime
	RuntimeMsgs *MsgHub `json:"-"`
}

// ControllerDID returns the current user did
func (ecs EdgeConfigSchema) ControllerDID() string {
	return fmt.Sprintf("did:cosmos:net:%s:%s", ecs.ChainID, ecs.ControllerDidID)
}

// MediatorDID returns the cloud agent mediator did
func (ecs EdgeConfigSchema) MediatorDID() string {
	return "did:cosmos:net:cosmoscash-testnet:mediatortestnetws10"
}

// CloudAgentAPIURL returns the cloud agent mediator did
func (ecs EdgeConfigSchema) CloudAgentAPIURL() string {
	return "https://agent.cosmos-cash.app.beta.starport.cloud"
}

func GetAppData(subPath ...string) (string, bool) {
	v := []string{"data"}
	v = append(v, subPath...)
	return GetAppConfig(v...)
}

func GetAppConfig(subPath ...string) (cfgPath string, exists bool) {
	cfgPath, err := os.UserConfigDir()
	if err != nil {
		log.Fatalln(err)
	}
	cfgPath = path.Join(cfgPath, "cosmos-cash-agent")
	for _, sp := range subPath {
		cfgPath = path.Join(cfgPath, sp)
	}
	exists = true
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		exists = false
	}
	return
}
