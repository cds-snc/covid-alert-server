MODULE := github.com/CovidShield/server

CMDS := key-submission key-retrieval monolith

PROTO_FILES := $(shell find proto -name '*.proto')
PROTO_FILES_WITH_RPC :=

PROTO_GO    := $(patsubst proto/%.proto,pkg/proto/%/proto.pb.go,$(PROTO_FILES))
PROTO_RB    := $(patsubst proto/%.proto,test/lib/protocol/%_pb.rb,$(PROTO_FILES))
RPC_RB      := $(patsubst proto/%.proto,test/lib/protocol/%_services_pb.rb,$(PROTO_FILES_WITH_RPC))

GOFILES     := $(shell find . -type f -name '*.go')
TEST_FILES  := $(shell find test -type f -name '*_test.rb')

RELEASE_BUILDS := $(patsubst %,build/release/%,$(CMDS))
DEBUG_BUILDS   := $(patsubst %,build/debug/%,$(CMDS))

GO              := go
GOFMT           := gofmt
PROTOC          := protoc
GCFLAGS         :=
GCFLAGS_RELEASE := $(GCFLAGS)
GCFLAGS_DEBUG   := $(GCFLAGS) -N -l

# You may need/want to set this to "bundle exec grpc_tools_ruby_protoc"
GRPC_TOOLS_RUBY_PROTOC := grpc_tools_ruby_protoc

BRANCH=`git rev-parse --abbrev-ref HEAD`
REVISION=`git rev-parse HEAD`
GOLDFLAGS="-X $(MODULE)/pkg/server.branch=$(BRANCH) -X $(MODULE)/pkg/server.revision=$(REVISION)"

default:  release
release:  $(PROTO_GO) $(RELEASE_BUILDS)
debug:    $(PROTO_GO) $(DEBUG_BUILDS)
all:      release debug $(PROTO_RB) $(RPC_RB)
proto:    $(PROTO_GO) $(PROTO_RB) $(RPC_RB)

build/debug/%: $(GOFILES) $(PROTO_GO)
	@mkdir -p "$(@D)"
	@echo "     \e[1;34mgo build \e[0;1m(debug)\e[0m  $@"
	@$(GO) build -trimpath -i -ldflags=$(GOLDFLAGS) -o "$@" -gcflags "$(GCFLAGS_DEBUG)" "$(MODULE)/cmd/$(@F)"

build/release/%: $(GOFILES) $(PROTO_GO)
	@mkdir -p "$(@D)"
	@echo "   \e[1;34mgo build \e[0;1m(release)\e[0m  $@"
	@$(GO) build -trimpath -i -ldflags=$(GOLDFLAGS) -o "$@" -gcflags "$(GCFLAGS_RELEASE)" "$(MODULE)/cmd/$(@F)"

pkg/proto/%/proto.pb.go: proto/%.proto
	@mkdir -p "$(@D)"
	@echo "          \e[1;34mprotoc \e[0;1m(go)\e[0m  $@"
	@$(PROTOC) --go_out=. "--proto_path=$(*D)" "$<"
	@mv "$(@D)/$(patsubst %.proto,%,$(*F)).pb.go" "$(@D)/proto.pb.go"

test/lib/protocol/%_pb.rb: proto/%.proto
	@mkdir -p "$(@D)"
	@echo "          \e[1;34mprotoc \e[0;1m(rb)\e[0m  $@"
	@grpc_tools_ruby_protoc -I ./proto --ruby_out=test/lib/protocol "$<"

test/lib/protocol/%_services_pb.rb: proto/%.proto
	@mkdir -p "$(@D)"
	@echo "          \e[1;34mprotoc \e[0;1m(rb)\e[0m  $@"
	@grpc_tools_ruby_protoc -I ./proto --grpc_out=test/lib/protocol "$<"

test: debug $(PROTO_RB) $(RPC_RB)
	ruby $(foreach file,$(TEST_FILES),-r ./$(file)) -e ''

clean:
	@echo "                   \e[1;31mrm\e[0m  $(CMDS)"
	@rm -f $(CMDS)
	@echo "                   \e[1;31mrm\e[0m  build"
	@rm -rf build

clean-proto:
	@echo "                   \e[1;31mrm\e[0m  $(PROTO_GO)"
	@rm -f $(PROTO_GO)
	@echo "                   \e[1;31mrm\e[0m  $(PROTO_RB)"
	@rm -f $(PROTO_RB)
	@echo "                   \e[1;31mrm\e[0m  $(RPC_RB)"
	@rm -f $(RPC_RB)

format:
	@$(GOFMT) -w .

.PHONY: all default release generate clean test proto clean-proto format
