package ui

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wealdtech/go-merkletree"

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
		case config.MsgUpdateContacts:
			contacts.Set(m.Payload.([]interface{}))
		case config.MsgUpdateContact:
			status := m.Payload.(string)
			messages.Append(status)

		case config.MsgTextReceived:
			tm := m.Payload.(model.TextMessage)
			msgHistory := state.Messages[tm.Channel]
			msgHistory = append(msgHistory, tm) // refresh view
			state.Messages[tm.Channel] = msgHistory

			// save the state
			appCfg.RuntimeMsgs.Notification <- config.NewAppMsg(config.MsgSaveState, nil)

			// append the message if on focus
			contact, err := getContact(state.SelectedContact)
			if err != nil {
				continue
			}
			if tm.From == appCfg.ControllerName {
				messages.Append(tm.String())
				// do not reprocess outoging messages
				break
			}
			if tm.Channel == contact.ConnectionID {
				messages.Append(tm.String())
			}

			if pr, isRequest := model.ParsePresentationRequest(tm.Content); isRequest {
				switch prT := pr.(type) {
				// FOR PAYMENT
				case *model.PaymentRequest:
					paymentReq := *prT
					// TODO: this should be now tunneled to the other party app
					// now the other party should receive this message and render confirmation dialog
					RenderRequestConfirmation(fmt.Sprintf("Payment request from %s", tm.From), paymentReq,
						func(pr model.PresentationRequest) {
							// TODO: send payment via token wallet
							log.WithFields(log.Fields{"recipient": paymentReq.Recipient}).Infoln("payment approved")
							appCfg.RuntimeMsgs.TokenWalletIn <- config.NewAppMsg(config.MsgPaymentRequest, paymentReq)
							appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgSendText,
								model.NewTextMessage(tm.Channel,
									appCfg.ControllerName,
									fmt.Sprintf("payment request accepted, you should receive %v%s shortly", paymentReq.Amount, paymentReq.Denom),
								))
						}, func(pr model.PresentationRequest) {
							// TODO: send a message on the chat that the request has been aborted
							log.WithFields(log.Fields{"recipient": paymentReq.Recipient}).Infoln("payment denied")
							appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgSendText,
								model.NewTextMessage(tm.Channel,
									appCfg.ControllerName,
									fmt.Sprintf("unfortunately your payment request of %v%s has been denied", paymentReq.Amount, paymentReq.Denom),
								))
						})
				case *model.RegulatorCredentialRequest:
					req := *prT
					appCfg.RuntimeMsgs.TokenWalletIn <- config.NewAppMsg(config.MsgIssueVC, req)
				case *model.RegistrationCredentialRequest:
					req := *prT
					RenderRequestConfirmation(fmt.Sprintf("Registration application from %s", tm.From), req,
						func(pr model.PresentationRequest) {
							apr := pr.(model.RegistrationCredentialRequest)
							c := model.NewRegistrationCredential(appCfg.ControllerDID(), req.SubjectDID, apr)
							appCfg.RuntimeMsgs.TokenWalletIn <- config.NewAppMsg(config.MsgIssueVC, c)
							appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgSendText,
								model.NewTextMessage(tm.Channel,
									appCfg.ControllerName,
									"Your registration has been accepted, check your credentials for confirmation",
								))
							appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgSendText,
								model.NewTextMessage(tm.Channel,
									appCfg.ControllerName,
									"Unfortunately your registration has been denied",
								))
						}, func(pr model.PresentationRequest) {
							appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgSendText,
								model.NewTextMessage(tm.Channel,
									appCfg.ControllerName,
									"Wme are sorry to inform you that the application has not been accepted.",
								))
						})
				case *model.LicenseCredentialRequest:
					req := *prT
					appCfg.RuntimeMsgs.TokenWalletIn <- config.NewAppMsg(config.MsgIssueVC, req)
				case *model.EMoneyApplicationRequest:
					req := *prT
					RenderRequestConfirmation(fmt.Sprintf("E-Money application from %s", tm.From), req,
						func(pr model.PresentationRequest) {
							apr := pr.(model.EMoneyApplicationRequest)
							c := model.NewPoKYCCredential(appCfg.ControllerDID(), apr.SubjectDID, apr)
							appCfg.RuntimeMsgs.TokenWalletIn <- config.NewAppMsg(config.MsgIssueVC, c)
							appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgSendText,
								model.NewTextMessage(tm.Channel,
									appCfg.ControllerName,
									"Your application has been accepted! welcome! (check your credentials for confirmation)",
								))
							appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgSendText,
								model.NewTextMessage(tm.Channel,
									appCfg.ControllerName,
									"Once you receive your credential, you can send a request for tokens",
								))
						}, func(pr model.PresentationRequest) {
							appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgSendText,
								model.NewTextMessage(tm.Channel,
									appCfg.ControllerName,
									"Wme are sorry to inform you that the application has not been accepted.",
								))
						})
				default:
					log.Errorln("unknown presentation request", prT)
				}

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

	contact, err := getContact(iID)
	if err != nil {
		log.Errorln("error retrieving contact id ", iID, err)
		return
	}
	msgHistory, found := state.Messages[contact.ConnectionID]
	if !found {
		msgHistory = make([]model.TextMessage, 0)
	}
	//contact, _ := state.Contacts[name]
	msgs := make([]string, len(msgHistory))
	for i, msg := range msgHistory {
		msgs[i] = msg.String()
	}
	messages.Set(msgs)
	// copy to clipboard the name
	appCfg.RuntimeMsgs.Notification <- config.NewAppMsg(config.MsgClipboard, contact.ConnectionID)
	// update selected contact index
	state.SelectedContact = iID
}

// executeCmd get executed every time the text input field get executed
func executeCmd() {
	val, _ := userCommand.Get()
	log.WithFields(log.Fields{"command": val}).Infoln("user command received")

	defer userCommand.Set("")

	// FINALLY RECORD THE MESSAGE IN THE CHAT
	// if no contact is selected move on
	//if state.SelectedContact < 0 {
	//	return
	//}

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
				contact, err := getContact(state.SelectedContact)
				if err != nil {
					break
				}
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
				contact, err := getContact(state.SelectedContact)
				if err != nil {
					break
				}
				payload := contact.ConnectionID + " " + s[3]
				appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgApproveInvitation, payload)
			case "create":
			case "c":
				appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgCreateInvitation, "")
			case "router":
			case "r":
				contact, err := getContact(state.SelectedContact)
				if err != nil {
					break
				}
				appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgCreateInvitation, contact.ConnectionID)
			}
		case "delete":
		case "d":
			contact, err := getContact(state.SelectedContact)
			if err != nil {
				break
			}
			messages.Set([]string{})
			state.Messages[contact.ConnectionID] = []model.TextMessage{}
			appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgDeleteConnection, contact.ConnectionID)
		case "mediator":
		case "m":
			contact, err := getContact(state.SelectedContact)
			if err != nil {
				break
			}
			appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgAddMediator, contact.ConnectionID)
		case "status":
		case "s":
			contact, err := getContact(state.SelectedContact)
			if err != nil {
				break
			}
			appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgGetConnectionStatus, contact.Connection.ConnectionID)
		}
	case "chain", "c":
		switch s[1] {
		case "address", "a":
			appCfg.RuntimeMsgs.TokenWalletIn <- config.NewAppMsg(config.MsgChainAddAddress, nil)
		case "payment-request", "pr":
			r := model.NewPaymentRequest("sEUR", "Payment for the services")
			appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgVCs, model.NewCallableEnvelope(nil, func(account string) {
				r.Recipient = account
				// render the payment request
				RenderPresentationRequest("Please enter the payment request details", r, func(i interface{}) {
					// when the payment request has been filled get the updated data
					contact, err := getContact(state.SelectedContact)
					if err != nil {
						return
					}
					tm := model.NewTextMessage(contact.ConnectionID, appCfg.ControllerName, helpers.ToJson(i))
					// route the message to the agent
					appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgSendText, tm)
				})
			}))
		}

	case "debug", "d":
		switch s[1] {
		case "payment-request", "pr":
			// TODO this should be retrieved from the aries credentials not from the chain wallet
			appCfg.RuntimeMsgs.TokenWalletIn <- config.NewAppMsg(config.MsgChainGetAddresses, model.NewCallableEnvelope(nil, func(addr string) {
				r := model.NewPaymentRequest("cash", "Payment for the services")
				r.Recipient = addr
				// render the payment request
				RenderPresentationRequest("Please enter the payment request details", r, func(i interface{}) {
					// when the payment request has been filled get the updated data
					contact, _ := getContact(state.SelectedContact)
					tm := model.NewTextMessage(contact.ConnectionID, appCfg.ControllerName, helpers.ToJson(i))
					// route the message to the agent
					appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgSendText, tm)
				})
			}))
		case "regulator-credential", "rg":
			r := model.NewRegulatorCredentialRequest(appCfg.ControllerDID())
			RenderPresentationRequest("Enter regulator data", r, func(i interface{}) {
				//appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgSendText, tm)

			})
		case "registration-credential", "rc":
			r := model.NewRegistrationCredentialRequest("EU")
			RenderPresentationRequest("Enter registration request", r, func(i interface{}) {
				contact, err := getContact(state.SelectedContact)
				if err != nil {
					return
				}
				tm := model.NewTextMessage(contact.ConnectionID, appCfg.ControllerName, helpers.ToJson(i))
				// route the message to the agent
				appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgSendText, tm)
			})
		case "license-credential", "lc":
			r := model.NewLicenseCredentialRequest("MICAEMI", "EU")
			RenderPresentationRequest("Enter license request", r, func(i interface{}) {
				// when the payment request has been filled get the updated data
				contact, err := getContact(state.SelectedContact)
				if err != nil {
					return
				}
				tm := model.NewTextMessage(contact.ConnectionID, appCfg.ControllerName, helpers.ToJson(i))
				// route the message to the agent
				appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgSendText, tm)
			})
		case "emoney-application", "ea":
			r := model.NewEMoneyApplicationRequest(appCfg.ControllerDID())
			RenderPresentationRequest("Enter E-Money request details", r, func(i interface{}) {
				rF := i.(model.EMoneyApplicationRequest)
				contact, err := getContact(state.SelectedContact)
				if err != nil {
					return
				}
				// calculate the "ZKP"
				data := [][]byte{
					[]byte(rF.Name),
					[]byte(rF.Surname),
					[]byte(rF.Age),
				}
				tree, _ := merkletree.NewUsing(data, vcTypes.New("whatever"), nil)
				rF.ZKP = hex.EncodeToString(tree.Root())
				// now send the stuff
				tm := model.NewTextMessage(contact.ConnectionID, appCfg.ControllerName, helpers.ToJson(rF))
				appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgSendText, tm)
				// for debug purposes:
				// appCfg.RuntimeMsgs.Notification <- config.NewAppMsg(config.MsgTextReceived, tm)
			})
		case "clear-credentials", "cc":
			appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgClearCredentials, nil)

		}

	default:
		if contact, err := getContact(state.SelectedContact); err == nil {
			tm := model.NewTextMessage(contact.ConnectionID, appCfg.ControllerName, val)
			appCfg.RuntimeMsgs.AgentWalletIn <- config.NewAppMsg(config.MsgSendText, tm)
			// also record the message locally
			appCfg.RuntimeMsgs.Notification <- config.NewAppMsg(config.MsgTextReceived, tm)
		}
	}

}

// getContact helper method to get a contact
func getContact(id int) (c model.Contact, err error) {
	if contacts.Length() == 0 {
		err = fmt.Errorf("contact list empty")
		return
	}
	if id >= contacts.Length() {
		err = fmt.Errorf("out of bounds")
		return
	}
	i, err := contacts.GetValue(id)
	if err != nil {
		return
	}
	c = i.(model.Contact)
	if c.Connection == nil {
		err = fmt.Errorf("out of bounds")
		return
	}
	return
}
