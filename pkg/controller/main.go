package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"

	de "github.com/hyperledger/aries-framework-go/pkg/controller/command/didexchange"
	ks "github.com/hyperledger/aries-framework-go/pkg/controller/command/kms"
	"github.com/hyperledger/aries-framework-go/pkg/controller/command/messaging"
)

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
	fmt.Printf("---> Request URL:\n %s\nPayload:\n%s\n", url, requestBody)
	fmt.Printf("<--- Reply:\n%s\n", bodyBytes)
}

func post(client *http.Client, url string, requestBody, val interface{}) {
	if requestBody != nil {
		request(client, "POST", url, bitify(requestBody), val)
	} else {
		request(client, "POST", url, nil, val)
	}

}
func get(client *http.Client, url string, val interface{}) {
	request(client, "GET", url, nil, val)
}

func bitify(in interface{}) io.Reader {
	v, err := json.Marshal(in)
	if err != nil {
		panic(err.Error())
	}
	return bytes.NewBuffer(v)
}

func main() {

	var (
		bobAgent   = "http://localhost:8090"
		bobDID     = "did:cosmos:net:cash:bob"
		aliceAgent = "http://localhost:7090"
		aliceDID   = "did:cosmos:net:cash:alice"
	)

	bobConnID, _ := DIDExchange(bobAgent, bobDID, aliceAgent, aliceDID)
	DIDMessaging(bobAgent, aliceAgent, bobConnID)

	print("yey!")
}

type genericInviteMsg struct {
	ID      string   `json:"@id"`
	Type    string   `json:"@type"`
	Purpose []string `json:"~purpose"`
	Message string   `json:"message"`
	From    string   `json:"from"`
}

func DIDMessaging(bobAgent, aliceAgent, connID string) {
	// DID Messaging
	// https://github.com/hyperledger/aries-framework-go/blob/main/docs/rest/openapi_demo.md#steps-for-custom-message-handling

	var (
		client = &http.Client{}
		reqURL string
	)

	var (
		createService messaging.RegisterMsgSvcArgs
		genericMsg    genericInviteMsg
		request       messaging.SendNewMessageArgs
	)

	// Messaging service
	reqURL = fmt.Sprint(aliceAgent, "/message/register-service")
	println("7. ALICE creates a service for BOB to send messages", reqURL)
	createService.Type = "https://didcomm.org/generic/1.0/message"
	createService.Purpose = []string{"meeting", "appointment", "event"}
	createService.Name = "generic-invite"

	var resp interface{}
	post(client, reqURL, createService, resp)

	reqURL = fmt.Sprint(aliceAgent, "/message/services")
	println("8. ALICE verifies the service has been created", reqURL)
	get(client, reqURL, resp)

	genericMsg.ID = "12123123213213"
	genericMsg.Type = "https://didcomm.org/generic/1.0/message"
	genericMsg.Purpose = []string{"meeting"}
	genericMsg.Message = "fight me you coward"
	genericMsg.From = "Bob"

	rawBytes, _ := json.Marshal(genericMsg)

	request.ConnectionID = connID
	request.MessageBody = rawBytes

	reqURL = fmt.Sprint(bobAgent, "/message/send")
	println("9. BOB sends a message of type generic invite to ALICE", reqURL)
	post(client, reqURL, request, resp)
}

func DIDExchange(bobAgent, bobDID, aliceAgent, aliceDID string) (string, string) {
	// DID Exchange
	// https://github.com/hyperledger/aries-framework-go/blob/main/docs/rest/openapi_demo.md#steps-for-didexchange

	var (
		client = &http.Client{}
		reqURL string
		params url.Values
	)

	var (
		keySetRsp   ks.CreateKeySetResponse
		connection  de.QueryConnectionResponse
		connections de.QueryConnectionsResponse
		invite      de.CreateInvitationResponse
		//implicitInvite de.ImplicitInvitationResponse
		receiveInvite   de.ReceiveInvitationResponse
		acceptInvite    de.AcceptInvitationResponse
		confirmExchange de.ExchangeResponse
	)

	println("DID Exchange")
	//x := de.ImplicitInvitationArgs{
	//	InviterDID:        aliceDID,
	//	InviterLabel:      "AliceAgent",
	//	InviteeDID:        bobDID,
	//	InviteeLabel:      "BobAgent",
	//}
	//
	//msg, err = json.Marshal(x)
	//if err != nil {
	//	panic(err)
	//}

	println("ALICE", aliceDID)
	println("BOB  ", bobDID)
	routerID := fmt.Sprint(rand.Int())
	println("router id", routerID)

	v, _ := base64.StdEncoding.DecodeString(keySetRsp.PublicKey)
	println("keyID", keySetRsp.KeyID)
	println("keyPub (base64)", keySetRsp.PublicKey)
	println("keyPub (hex)", hex.EncodeToString(v))

	// Create invitation
	params = url.Values{}
	params.Add("public", aliceDID)
	reqURL = fmt.Sprint(aliceAgent, "/connections/create-invitation?public=", aliceDID, "&label=AliceAgent")
	println("1. ALICE creates an invitation", reqURL)
	post(client, reqURL, nil, &invite)

	//params = url.Values{}
	//params.Add("their_did", aliceDID)
	//params.Add("their_label", "AliceAgent")
	//params.Add("their_did", bobDID)
	//params.Add("their_did", "BobAgent")
	//reqURL = fmt.Sprint(aliceAgent, "/connections/create-implicit-invitation?", params.Encode())
	//println("1. ALICE creates an implicit invitation", reqURL)
	//post(client, reqURL, nil, &implicitInvite)

	reqURL = fmt.Sprint(bobAgent, "/connections/receive-invitation")
	println("2. BOB receive the invitation", reqURL)
	post(client, reqURL, invite.Invitation, &receiveInvite)

	// Check connection
	reqURL = fmt.Sprint(bobAgent, "/connections/", receiveInvite.ConnectionID)
	println("3. BOB inspect the invitation", reqURL)
	get(client, reqURL, &connection)

	// Check connection
	reqURL = fmt.Sprint(bobAgent, "/connections")
	println("4. BOB lists connections ", reqURL)
	get(client, reqURL, &connections)

	reqURL = fmt.Sprint(bobAgent, "/connections/", receiveInvite.ConnectionID, "/accept-invitation")
	println("5. BOB accepts the connection", reqURL)
	//var accept de.AcceptInvitationResponse
	post(client, reqURL, nil, &acceptInvite)

	// Check connection
	reqURL = fmt.Sprint(aliceAgent, "/connections")
	println("6. ALICE lists connections", reqURL)
	get(client, reqURL, &connections)

	var aliceConnID string
	for _, c := range connections.Results {
		if c.State == "requested" {
			reqURL = fmt.Sprint(aliceAgent, "/connections/", c.ConnectionID, "/accept-request")
			println("7. ALICE accepts the connection request (replied from bob)", reqURL)
			post(client, reqURL, nil, &confirmExchange)

			reqURL = fmt.Sprint(bobAgent, "/connections/", receiveInvite.ConnectionID)
			println("8.1 BOB get connection", receiveInvite.ConnectionID)
			//var accept de.AcceptInvitationResponse
			get(client, reqURL, &connection)
			println("8.1 Connection state", connection.Result.State)

			reqURL = fmt.Sprint(aliceAgent, "/connections/", c.ConnectionID)
			println("8.2 ALICE get connection", c.ConnectionID)
			aliceConnID = c.ConnectionID
			//var accept de.AcceptInvitationResponse
			get(client, reqURL, &connection)
			println("8.2 Connection state", connection.Result.State)
		}
	}

	return receiveInvite.ConnectionID, aliceConnID
}
