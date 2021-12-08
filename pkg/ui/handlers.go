package ui

import (
	"encoding/json"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/allinbits/cosmos-cash-agent/pkg/config"
	"github.com/allinbits/cosmos-cash-agent/pkg/helpers"
	"github.com/allinbits/cosmos-cash-agent/pkg/model"
	vcTypes "github.com/allinbits/cosmos-cash/v2/x/verifiable-credential/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/hyperledger/aries-framework-go/pkg/doc/verifiable"
	log "github.com/sirupsen/logrus"
)

// this section contains all the databindings to the ui
var (
	appCfg *config.EdgeConfigSchema
	state  *State
	// balances
	balances             = binding.NewStringList()
	balancesChainOfTrust = binding.NewString()
	// Credentials tab
	publicCredentials  = binding.NewStringList()
	privateCredentials = binding.NewStringList()
	credentialData     = binding.NewString()
	// Messages tab
	contacts    = binding.NewUntypedList()
	userCommand = binding.NewString()
	messages    = binding.NewStringList()
	// contactData = binding.NewString()
	// Marketplace tab
	marketplaces    = binding.NewStringList()
	marketplaceData = binding.NewString()
	// logs
	logData = binding.NewStringList()
)

// dispatcher this reads notifications and updates the
// data binding
func dispatcher(window fyne.Window, in chan config.AppMsg) {
	state = NewState()
	// first load the state
	// write the state on file
	statePath, exists := config.GetAppData("state.json")
	if !exists {
		helpers.WriteJson(statePath, state)

	}

	// now load the state
	helpers.LoadJson(statePath, state)

	// TODO where do we store connections?
	// now show the contacts
	//var contactNames []string
	//for k, _ := range state.Contacts {
	//	contactNames = append(contactNames, k)
	//}
	//contacts.Set(contactNames)

	// now handle the incoming notifications
	for {
		m := <-in
		switch m.Typ {
		case config.MsgClipboard:
			window.Clipboard().SetContent(m.Payload.(string))
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
			publicCredentials.Set(credentialIDs)
		case config.MsgVCs:
			// populate private credentials
			var credentialIDs []string
			for _, c := range m.Payload.([]verifiable.Credential) {
				credentialIDs = append(credentialIDs, c.ID)
			}
			privateCredentials.Set(credentialIDs)
		case config.MsgVCData, config.MsgPublicVCData:
			if m.Payload == nil {
				credentialData.Set(string("No data"))
				continue
			}
			data, _ := json.MarshalIndent(m.Payload, "", " ")
			credentialData.Set(string(data))
		case config.MsgMarketplaces:
			var mks []string
			for _, c := range m.Payload.([]vcTypes.VerifiableCredential) {
				mks = append(mks, c.GetId())
			}
			marketplaces.Set(mks)
		//case config.MsgContactAdded:
		//	newContact := m.Payload.(*didexchange.Connection)
		//	contact := model.NewContact(*newContact)
		//	state.Contacts[contact.Connection.ConnectionID] = contact
		//	// update the model
		//	contacts.Append(newContact.ConnectionID)
		//	// request state save
		//	//appCfg.RuntimeMsgs.Notification <- config.NewAppMsg(config.MsgSaveState, nil)

		case config.MsgUpdateContacts:
			contacts.Set(m.Payload.([]interface{}))
		case config.MsgUpdateContact:
			status := m.Payload.(string)
			messages.Append(status)

		case config.MsgTextReceived:
			tm := m.Payload.(model.TextMessage)
			contact, _ := state.Contacts[tm.Channel]
			contact.Texts = append(contact.Texts, tm) // refresh view
			state.Contacts[tm.Channel] = contact

			if pr, isRequest := model.ParsePresentationRequest(tm.Content); isRequest {
				switch prT := pr.(type) {
				// FOR PAYMENT
				case *model.PaymentRequest:
					paymentReq := *prT
					// TODO: this should be now tunneled to the other party app
					// now the other party should receive this message and render confirmation dialog
					RenderRequestConfirmation("You got mail!", paymentReq,
						func(pr model.PresentationRequest) {
							// TODO: send payment via token wallet
							log.WithFields(log.Fields{"recipient": paymentReq.Recipient}).Infoln("payment approved")
							appCfg.RuntimeMsgs.TokenWalletIn <- config.NewAppMsg(config.MsgPaymentRequest, paymentReq)
						}, func(pr model.PresentationRequest) {
							// TODO: send a message on the chat that the request has been aborted
							log.WithFields(log.Fields{"recipient": paymentReq.Recipient}).Infoln("payment refused ")
						})
				case *model.RegulatorCredentialRequest:
					req := *prT
					appCfg.RuntimeMsgs.TokenWalletIn <- config.NewAppMsg(config.MsgIssueVC, req)
				case *model.RegistrationCredentialRequest:
					req := *prT
					appCfg.RuntimeMsgs.TokenWalletIn <- config.NewAppMsg(config.MsgIssueVC, req)
				case *model.LicenseCredentialRequest:
					req := *prT
					appCfg.RuntimeMsgs.TokenWalletIn <- config.NewAppMsg(config.MsgIssueVC, req)
				case *model.UserCredentialRequest:
					req := *prT
					appCfg.RuntimeMsgs.TokenWalletIn <- config.NewAppMsg(config.MsgIssueVC, req)

				default:
					log.Errorln("unknown presentation request", prT)
				}

			}

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

// publicCredentialSelected gets triggered when an item is selected in the public credential list
func publicCredentialSelected(iID widget.ListItemID) {
	v, _ := publicCredentials.GetValue(iID)
	log.Debugln("public credential selected", v)
	appCfg.RuntimeMsgs.TokenWalletIn <- config.NewAppMsg(config.MsgPublicVCData, v)
}

// privateCredentialSelected gets triggered when an item is selected in the privateCredentialList list
func privateCredentialSelected(iID widget.ListItemID) {
	v, _ := privateCredentials.GetValue(iID)
	log.Debugln("private credential selected", v)
	// TODO: this should be handled by ARIES
	appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgVCData, v)
}

// marketplacesSelected gets triggered when an item is selected in the credential list
func marketplacesSelected(iID widget.ListItemID) {
	v, _ := marketplaces.GetValue(iID)
	log.Debugln("marketplace selected", v)
	appCfg.RuntimeMsgs.TokenWalletIn <- config.NewAppMsg(config.MsgMarketplaceData, v)
}

// contactSelected gets triggered when an item is selected in the contact list
func contactSelected(iID widget.ListItemID) {
	contactData, err := contacts.GetValue(iID)
	if err != nil {
		log.Errorln("error retrieving contact id ", iID, err)
		return
	}
	contact := contactData.(model.Contact)
	//contact, _ := state.Contacts[name]
	msgs := make([]string, len(contact.Texts))
	for i, msg := range contact.Texts {
		msgs[i] = msg.String()
	}
	messages.Set(msgs)
	// copy to clipboard the name
	appCfg.RuntimeMsgs.Notification <- config.NewAppMsg(config.MsgClipboard, contact.Connection.ConnectionID)
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
			case "final":
			case "f":
				contact, _ := getContact(state.SelectedContact)
				payload := contact.Connection.ConnectionID + " " + s[3]
				appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgApproveRequest, payload)
			case "handle":
			case "h":
				payload := "{}" // empty json
				if len(s) > 3 {
					payload = s[3]
				}
				appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgHandleInvitation, payload)
			case "approve":
			case "a":
				contact, _ := getContact(state.SelectedContact)
				payload := contact.Connection.ConnectionID + " " + s[3]
				appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgApproveInvitation, payload)
			case "create":
			case "c":
				appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgCreateInvitation, "")
			case "router":
			case "r":
				contact, _ := getContact(state.SelectedContact)
				appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgCreateInvitation, contact.Connection.ConnectionID)
			}
		case "delete":
		case "d":
			contact, _ := getContact(state.SelectedContact)
			appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgDeleteConnection, contact.ConnectionID)
		case "mediator":
		case "m":
			contact, _ := getContact(state.SelectedContact)
			appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgAddMediator, contact.Connection.ConnectionID)
		case "status":
		case "s":
			contact, _ := getContact(state.SelectedContact)
			appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgGetConnectionStatus, contact.Connection.ConnectionID)
		case "chat":
		case "c":
			contact, _ := getContact(state.SelectedContact)
			tm := model.NewTextMessage(contact.Connection.ConnectionID, appCfg.ControllerName, s[2])
			appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgSendText, tm)
		}
	case "chain", "c":
		switch s[1] {
		case "address", "a":
			appCfg.RuntimeMsgs.TokenWalletIn <- config.NewAppMsg(config.MsgChainAddAddress, nil)
		}

		//hub.Notification <- "chain"

	case "debug", "d":
		switch s[1] {
		case "payment-request", "pr":
			r := model.NewPaymentRequest("sEUR", "Payment for the services")
			// TODO: the recipientAddress is selected from the credentials
			// in this case is the account used for the resolver demo did. see github.com/allinbits/cosmos-cash-misc
			r.Recipient = "cosmos1lcc4s4qkv0yg7ntmws6v44z5r5py27hap7pfw3"
			// render the payment request
			RenderPresentationRequest("Please enter the payment request details", r, func(i interface{}) {
				// when the payment request has been filled get the updated data
				contact, _ := getContact(state.SelectedContact)
				tm := model.NewTextMessage(contact.Connection.ConnectionID, appCfg.ControllerName, helpers.ToJson(i))
				// route the message to the agent
				appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgSendText, tm)
			})
		case "regulator-credential", "rg":
			r := model.NewRegulatorCredentialRequest(appCfg.ControllerDID())
			RenderPresentationRequest("Enter regulator data", r, func(i interface{}) {
				//appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgSendText, tm)

			})
		case "registration-credential", "rc":
			r := model.NewRegistrationCredentialRequest("EU")
			RenderPresentationRequest("Enter registration request", r, func(i interface{}) {
				contact, _ := getContact(state.SelectedContact)
				tm := model.NewTextMessage(contact.Connection.ConnectionID, appCfg.ControllerName, helpers.ToJson(i))
				// route the message to the agent
				appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgSendText, tm)
			})
		case "license-credential", "lc":
			r := model.NewLicenseCredentialRequest("MICAEMI", "EU")
			RenderPresentationRequest("Enter license request", r, func(i interface{}) {
				// when the payment request has been filled get the updated data
				contact, _ := getContact(state.SelectedContact)
				tm := model.NewTextMessage(contact.Connection.ConnectionID, appCfg.ControllerName, helpers.ToJson(i))
				// route the message to the agent
				appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgSendText, tm)
			})
		}
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

// getContact helper method to get a contact
func getContact(id int) (c model.Contact, err error) {
	i, err := contacts.GetValue(id)
	if err != nil {
		return
	}
	c = i.(model.Contact)
	return
}
