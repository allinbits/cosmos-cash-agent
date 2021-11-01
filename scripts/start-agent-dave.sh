#!/bin/bash

echo "Starting dave agent on port: 11090"
#go run ../cmd/elesto-agent/main.go start \
dlv debug ../cmd/elesto-agent/main.go -- start \
	--api-host localhost:11090 \
	--inbound-host http@localhost:11091 \
	--inbound-host-external http@http://localhost:11091 \
	--webhook-url http://localhost:7082/wh/dave \
	--agent-default-label DaveAgent \
	--database-type mem \
	--database-prefix dave \
	--log-level DEBUG \
	--http-resolver-url cosmos@http://localhost:2109/identifier/aries/

#
#
#

#go run ../cmd/elesto-agent/main.go start \
#	--api-host localhost:7090 \  <-- api endpoint must be public --> agent-001.cosmos-cash.beta.starport.net agent-007
#	--inbound-host http@localhost:7091,ws@localhost:7092 \  <-- STILL NOT SURE  
#	--inbound-host-external http@https://example.com:7091,ws@ws://localhost:7092 \ <-- NOT SURE
#	--webhook-url http://localhost:7082 \ <-- must be public agent-001-webook..... 
#	--agent-default-label AliceAgent \ 
#	--database-type mem \ 
#	--http-resolver-url cosmos@http://localhost:2109/identifier/ &  <-- RESOLVER URL (INTERNAL URL)
