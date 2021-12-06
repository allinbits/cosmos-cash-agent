package config

// MsgHub contains channels used by the components to send messages to each others
type MsgHub struct {
	Notification  chan AppMsg
	AgentWalletIn chan AppMsg
	TokenWalletIn chan AppMsg
}

func NewMsgHub() *MsgHub {
	return &MsgHub{
		Notification:  make(chan AppMsg, 4096),
		AgentWalletIn: make(chan AppMsg, 4096),
		TokenWalletIn: make(chan AppMsg, 4096),
	}
}

const (
	// MsgBalances from chain to ui with a list of update balances
	MsgBalances = iota
	// MsgDidDoc from chain to ui with a did document (at startup)
	MsgDidDoc
	// MsgChainOfTrust returns the list of verifiable credentials for a DENOM
	MsgChainOfTrust
	// MsgPublicVCs returns the list of verifiable credentials
	MsgPublicVCs
	// MsgPublicVCData returns the details of a verifiable credential
	MsgPublicVCData
	// MsgMarketplaces used for marketplace listing
	MsgMarketplaces
	// MsgMarketplaceData used for details of a marketplace
	MsgMarketplaceData
	// MsgVCs returns the list of verifiable credentials
	MsgVCs
	// MsgVCData returns the details of a verifiable credential
	MsgVCData
	// MsgIssueVC add private verifiable credential to the wallet
	MsgIssueVC
	//MsgContactAdded  used when a new contact si added
	MsgContactAdded
	// MsgUpdateContacts is used when updating all contacts in the list called every 30 seconds
	// is updated by the aries connection data store
	MsgUpdateContacts
	// MsgUpdateContact used when updating a connection by connection ID
	MsgUpdateContact
	// MsgTextReceived used when receiving messages
	MsgTextReceived
	// MsgSendText used to send text messages
	MsgSendText
	// MsgSaveState persist state to the disk
	MsgSaveState
	// MsgCreateInvitation creates an invitation to be used in another aries client
	MsgCreateInvitation
	// MsgHandleInvitation handles a DIDExchange invitation
	MsgHandleInvitation
	// MsgApproveInvitation approves an invitation needed for edge to edge mediator/routing connections
	MsgApproveInvitation
	// MsgApproveRequest approves a request for edge to edge mediator/routing connections
	MsgApproveRequest
	// MsgAddMediator adds a contact as a mediator this enables message routing between edge clients
	MsgAddMediator
	// MsgGetConnectionStatus gets the connection status of a contact
	MsgGetConnectionStatus
	// MsgDIDAddVerificationMethod add a verification method to a DID
	MsgDIDAddVerificationMethod
	// MsgChainAddAddress generate a new address and expose it as a verifiable credential
	MsgChainAddAddress
	// MsgSSIAddVC add a vc to the ssi wallet
	MsgSSIAddVC
	// MsgPaymentRequest receive a payment request
	MsgPaymentRequest
)

// AppMsg are messages that are exchanged within the app
type AppMsg struct {
	Typ     int
	Payload interface{}
}

func NewAppMsg(typ int, payload interface{}) AppMsg {
	return AppMsg{
		Typ:     typ,
		Payload: payload,
	}
}
