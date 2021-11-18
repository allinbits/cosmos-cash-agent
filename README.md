## Cosmos Cash controllers

The [Cosmos Cash Protocol](https://github.com/allinbits/cosmos-cash) is a Decentralized Key Management System (DKMS) which needs agents to implement certain protocols to allow the project to leverage Self sovereign identity [SSI](https://en.wikipedia.org/wiki/Self-sovereign_identity)

### Key Characteristics

- A [blockchain interface layer](https://github.com/allinbits/cosmos-cash-resolver) (known as a resolver) for creating and signing blockchain transactions.
- A resolver can be seen as part of a larger component known as the [VDR](https://github.com/allinbits/cosmos-cash):
  > Aries Verifiable Data Registry Interface: An interface for verifying data against an underlying ledger.
- A cryptographic wallet that can be used for secure storage of cryptographic secrets and other information (the secure storage tech, not a UI) used to build blockchain clients.
- An encrypted messaging system for allowing off-ledger interaction between those clients using multiple transport protocols.
- An implementation of ZKP-capable W3C verifiable credentials using the ZKP primitives found in Ursa.
- An implementation of the Decentralized Key Management System (DKMS) specification currently being incubated in under the name Cosmos Cash by AllinBits.
- A mechanism to build higher-level protocols and API-like use cases based on the secure messaging functionality.

#### Protocols

The following protocols are needed in the Cosmos Cash project.

#### 1. DIDExchange Protocol

#### 2. Introduce Protocol

#### 3. IssueCredential Protocol

#### 4. KMS

#### 5. Mediator

#### 6. Messaging

#### 7. OutOfBand Protocol

#### 8. PresentProof Protocol

#### 9. VDR

#### 10. Verifiable

### How to run

- `./scripts/start-agent.sh`


## Agent Webhook Relayer

The agent webhook relayer is a simple rest service that collects webhooks messages from agents
and makes them available for an edger agent or a controller to fetch them asynchronously.

The relayer exposes the following endpoints:

### Agent Webhook Receiver

This endpoint should be used in an Aries startup option `--webhook-url`

```
POST /ws/:agent

:msg_body
```

- the `:agent` path parameter is used to group the messages from the same agent.
- the `:msg_body`

#### Example

In this example we are running Bob Aries agent, to collect Aries webhook messages for Bob
we will configure the agent as follows:

```
aries start \
	--api-host localhost:7090 \
	--inbound-host http@localhost:7091 \
	--inbound-host-external http@http://localhost:7091 \
	--webhook-url http://localhost:2110/wh/bob \         <- this is the webook url for agent
	--agent-default-label BobAgent \
	--database-type leveldb \
	--database-prefix alice \
	--log-level DEBUG \
	--http-resolver-url cosmos@http://localhost:2109/identifier/aries/
```

### Agent Webhook Retriever

This endpoint should be used by a controller or an edge agent to retrieve the messages that
have been collected by the relayer for an agent.

It returns a list of webhook messages, note that once the messages are read they are removed
from the relayer.

```
POST /messages/:agent

[
  {
    "@id": "...."
  },
  ...
]
```

### How to run

```sh
go run cmd/webook-relayer/main.go -h                                                                                                                         │~

Usage of main.go:                                                                                                                │~
  -listen string                                                                                                                                               │~
        server listen address (default ":2110")                                                                                                                │~
  -n int                                                                                                                                                       │~
        Max number of messages that are kept per agent (default 4096)
  -x int                                                                                                                                                       │~
        Max-Requests-Per-Seconds: define the throttle limit in requests per seconds (default 10)

```
