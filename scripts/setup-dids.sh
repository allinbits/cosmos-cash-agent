#!/bin/bash

#cosmos-cashd tx did create-did alice --from alice --chain-id cash
#sleep 5
#cosmos-cashd tx did link-aries-agent alice http://localhost:7090 http://localhost:7091 --from alice --chain-id cash
#sleep 5
#cosmos-cashd query did dids --output json | jq
#
#
#cosmos-cashd tx did create-did bob --from bob --chain-id cash
#sleep 5
#cosmos-cashd tx did link-aries-agent bob http://localhost:8090 http://localhost:8091 --from bob --chain-id cash
#sleep 5
#cosmos-cashd query did dids --output json | jq

OPTS="--node https://cosmos-cash.app.beta.starport.cloud:443 --chain-id cosmoscash-testnet"
FAUCET=https://faucet.cosmos-cash.app.beta.starport.cloud
MEDIATOR=mediatortestnetws1

cosmos-cashd keys add $MEDIATOR
curl -X POST -d "{\"address\": \"$(cosmos-cashd keys show $MEDIATOR -a)\"}" $FAUCET
sleep 4
cosmos-cashd tx did create-did $MEDIATOR --from $MEDIATOR $OPTS -y
sleep 2
cosmos-cashd query did did did:cosmos:net:cosmoscash-testnet:$MEDIATOR $OPTS
sleep 3
cosmos-cashd tx did link-aries-agent $MEDIATOR http://localhost:8090 ws://localhost:8092 --from $MEDIATOR $OPTS -y
