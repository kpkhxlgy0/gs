#!/bin/sh

##################################################
###   proto & api
##################################################
#modify the path to your self path.
#export PATH_AGENT=/go/gonet2/agent/src/client_handler
#export PATH_GAME=/go/gonet2/game/src/client_handler
export PATH_AGENT=./proto_code/agent/
export PATH_GAME=./proto_code/game/
export PATH_CLIENT=./proto_code/client/
go get github.com/codegangsta/cli

## api.txt
go run api.go --min 0 --max 1000 > $PATH_AGENT/api.go; go fmt $PATH_AGENT/api.go
go run api.go --min 1001 --max 32767 > $PATH_GAME/api.go; go fmt $PATH_GAME/api.go
go run api.go --template "templates/client/api.tmpl"  --min 0 --max 32767 > $PATH_CLIENT/NetApi.cs

## proto.txt
go run proto.go > $PATH_AGENT/proto.go; go fmt $PATH_AGENT/proto.go
go run proto.go > $PATH_GAME/proto.go; go fmt $PATH_GAME/proto.go
go run proto.go --template "templates/client/proto.tmpl" --binding "cs" > $PATH_CLIENT/NetProto.cs
