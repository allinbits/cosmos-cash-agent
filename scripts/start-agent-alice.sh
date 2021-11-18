#!/bin/bash
echo "Starting alice agent on port: 7090"

dlv debug ../cmd/elesto-agent/main.go -- start \
	--api-host localhost:7090 \
	--inbound-host http@localhost:7091 \
	--inbound-host-external http@http://localhost:7091 \
	--outbound-transport ws \
	--webhook-url http://localhost:7082/wh/alice \
	--auto-accept true \
	--transport-return-route all \
	--agent-default-label AliceAgent \
	--database-type mem \
	--database-prefix alice \
	--log-level DEBUG \
	--http-resolver-url cosmos@http://localhost:2109/identifier/aries/
