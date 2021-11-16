package ui

import (
	"encoding/json"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/allinbits/cosmos-cash-agent/pkg/config"
	"github.com/allinbits/cosmos-cash-agent/pkg/helpers"
	"github.com/allinbits/cosmos-cash-agent/pkg/model"
	vcTypes "github.com/allinbits/cosmos-cash/v2/x/verifiable-credential/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	log "github.com/sirupsen/logrus"
	"strings"

	"github.com/hyperledger/aries-framework-go/pkg/client/didexchange"
)

// this section contains all the databindings to the ui
var (
	appCfg *config.EdgeConfigSchema
	state  *State
	// balances
	balances             = binding.NewStringList()
	balancesChainOfTrust = binding.NewString()
	// Credentials tab
	// TODO need to have separate stuff for public and private credentials
	credentials    = binding.NewStringList()
	credentialData = binding.NewString()
	// Messages tab
	contacts    = binding.NewStringList()
	userCommand = binding.NewString()
	messages    = binding.NewStringList()
	// contactData = binding.NewString()
	// Marketplace tab
	marketplaces    = binding.NewStringList()
	marketplaceData = binding.NewString()
	// logs
	logData = binding.NewString()
)

// dispatcher this reads notifications and updates the
// data binding
func dispatcher(in chan config.AppMsg) {
	state = NewState()
	// first load the statate
	// write the state on file
	statePath, exists := config.GetAppData("state.json")
	if !exists {
		helpers.WriteJson(statePath, state)
	}
	// now load the state
	helpers.LoadJson(statePath, state)
	// now show the contacts
	var contactNames []string
	for k, _ := range state.Contacts {
		contactNames = append(contactNames, k)
	}
	contacts.Set(contactNames)

	// now handle the incoming notifications
	for {
		m := <-in
		switch m.Typ {
		case config.MsgSaveState:
			log.Debugln("saving state to file", statePath)
			helpers.WriteJson(statePath, state)
		case config.MsgBalances:
			// populate the list of balances
			var newBalances []string
			for _, c := range m.Payload.(sdk.Coins) {
				newBalances = append(newBalances, c.String())
			}
			balances.Set(newBalances)
		case config.MsgChainOfTrust:
			// populate the chain of trust for a denom
			vcs := m.Payload.([]vcTypes.VerifiableCredential)
			data, _ := json.MarshalIndent(vcs, "", " ")
			balancesChainOfTrust.Set(string(data))
		case config.MsgPublicVCs:
			// populate public credentials
			var credentialIDs []string
			for _, c := range m.Payload.([]vcTypes.VerifiableCredential) {
				credentialIDs = append(credentialIDs, c.Id)
			}
			credentials.Set(credentialIDs)
		case config.MsgPublicVCData:
			if m.Payload == nil {
				credentialData.Set(string("No data"))
				continue
			}
			vcs := m.Payload.(vcTypes.VerifiableCredential)
			data, _ := json.MarshalIndent(vcs, "", " ")
			credentialData.Set(string(data))
		case config.MsgMarketplaces:
			var mks []string
			for _, c := range m.Payload.([]vcTypes.VerifiableCredential) {
				mks = append(mks, c.GetId())
			}
			marketplaces.Set(mks)
		case config.MsgContactAdded:
			newContact := m.Payload.(*didexchange.Connection)

			contact := model.NewContact(newContact.TheirLabel, newContact.TheirDID, *newContact)
			state.Contacts[contact.Connection.ConnectionID] = contact
			// update the model
			contacts.Append(newContact.ConnectionID)

			// request state save
			//appCfg.RuntimeMsgs.Notification <- config.NewAppMsg(config.MsgSaveState, nil)

		case config.MsgUpdateContacts:
			allContacts := m.Payload.([]*didexchange.Connection)
			cons, _ := contacts.Get()
			for _, contact := range allContacts {
				if !stringInSlice(contact.ConnectionID, cons) {
					contacts.Append(contact.ConnectionID)
				}
			}
		case config.MsgUpdateContact:
			status := m.Payload.(string)
			messages.Append(status)

		case config.MsgTextReceived:
			tm := m.Payload.(model.TextMessage)
			contact, _ := state.Contacts[tm.Channel]
			contact.Texts = append(contact.Texts, tm) // refresh view
			state.Contacts[tm.Channel] = contact

			// save the state
			appCfg.RuntimeMsgs.Notification <- config.NewAppMsg(config.MsgSaveState, nil)

			// append the message if on focus
			channel, _ := contacts.GetValue(state.SelectedContact)
			if tm.Channel == channel || tm.From == appCfg.ControllerName {
				messages.Append(tm.String())
			}
		}
	}
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// balancesSelected gets triggered when an item is selected in the balance list
func balancesSelected(iID widget.ListItemID) {
	v, _ := balances.GetValue(iID)
	log.Debugln("token selected", v)
	appCfg.RuntimeMsgs.TokenWalletIn <- config.NewAppMsg(config.MsgChainOfTrust, v)
}

// credentialsSelected gets triggered when an item is selected in the credential list
func credentialsSelected(iID widget.ListItemID) {
	v, _ := credentials.GetValue(iID)
	log.Debugln("credential selected", v)
	appCfg.RuntimeMsgs.TokenWalletIn <- config.NewAppMsg(config.MsgPublicVCData, v)
}

// marketplacesSelected gets triggered when an item is selected in the credential list
func marketplacesSelected(iID widget.ListItemID) {
	v, _ := marketplaces.GetValue(iID)
	log.Debugln("marketplace selected", v)
	appCfg.RuntimeMsgs.TokenWalletIn <- config.NewAppMsg(config.MsgMarketplaceData, v)
}

// contactSelected gets triggered when an item is selected in the contact list
func contactSelected(iID widget.ListItemID) {
	name, _ := contacts.GetValue(iID)
	contact, _ := state.Contacts[name]
	msgs := make([]string, len(contact.Texts))
	for i, msg := range contact.Texts {
		msgs[i] = msg.String()
	}
	messages.Set(msgs)
	// update selected contact index
	state.SelectedContact = iID
}

// executeCmd get executed every time the text input field get executed
func executeCmd() {
	val, _ := userCommand.Get()
	log.WithFields(log.Fields{"command": val}).Infoln("user command received")

	// parse the command
	s := strings.Split(val, " ")

	switch s[0] {
	case "ssi":
	case "s":
		switch s[1] {
		case "invitation":
		case "i":
			switch s[2] {
			case "handle":
			case "h":
				appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgHandleInvitation, s[3])
			case "approve":
			case "a":
				contact, _ := contacts.GetValue(state.SelectedContact)
				//		if len(s) > 3 {
				//			appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgApproveInvitation, contact+" "+s[3])
				//		}
				payload := contact + " " + s[3]
				appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgApproveInvitation, payload)
			case "create":
			case "c":
				appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgCreateInvitation, "")
			case "router":
			case "r":
				contact, _ := contacts.GetValue(state.SelectedContact)
				appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgCreateInvitation, contact)
			}
		case "mediator":
		case "m":
			contact, _ := contacts.GetValue(state.SelectedContact)
			appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgAddMediator, contact)
		case "status":
		case "s":
			contact, _ := contacts.GetValue(state.SelectedContact)
			appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgGetConnectionStatus, contact)

		}
	case "chain":
		//hub.Notification <- "chain"
	}

	// FINALLY RECORD THE MESSAGE IN THE CHAT
	// if no contact is selected move on
	if state.SelectedContact < 0 {
		return
	}
	//channel, _ := contacts.GetValue(state.SelectedContact)
	// send it as a text received
	//	appCfg.RuntimeMsgs.Notification <- config.NewAppMsg(
	//		config.MsgTextReceived,
	//		model.NewTextMessage(channel, appCfg.ControllerName, val),
	//	)
	// reset the command
	userCommand.Set("")
}
