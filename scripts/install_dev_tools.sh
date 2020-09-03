#!/bin/bash
export PATH=/usr/local/go/bin:$PATH
export GOPATH=/root/go
export GOBIN=/usr/local/bin 
export GO111MODULE=on

bundle install 

go install google.golang.org/protobuf/cmd/protoc-gen-go

cd /

go mod init foo/bar

GOPATH=/root/go go get -u github.com/mdempsky/gocode
GOPATH=/root/go go get -u github.com/uudashr/gopkgs/v2/cmd/gopkgs
GOPATH=/root/go go get -u github.com/ramya-rao-a/go-outline
GOPATH=/root/go go get -u github.com/acroca/go-symbols
GOPATH=/root/go go get -u golang.org/x/tools/cmd/guru
GOPATH=/root/go go get -u golang.org/x/tools/cmd/gorename
GOPATH=/root/go go get -u github.com/cweill/gotests/...
GOPATH=/root/go go get -u github.com/fatih/gomodifytags
GOPATH=/root/go go get -u github.com/josharian/impl
GOPATH=/root/go go get -u github.com/davidrjenni/reftools/cmd/fillstruct
GOPATH=/root/go go get -u github.com/haya14busa/goplay/cmd/goplay
GOPATH=/root/go go get -u github.com/godoctor/godoctor
GOPATH=/root/go go get -u github.com/go-delve/delve/cmd/dlv
GOPATH=/root/go go get -u github.com/stamblerre/gocode
GOPATH=/root/go go get -u github.com/rogpeppe/godef
GOPATH=/root/go go get -u golang.org/x/tools/cmd/goimports
GOPATH=/root/go go get -u golang.org/x/lint/golint
GOPATH=/root/go go get -u github.com/stamblerre/gocode 
