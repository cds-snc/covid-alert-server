MODULE := github.com/CovidShield/backend

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
PROTOC          := protoc
GCFLAGS         :=
GCFLAGS_RELEASE := $(GCFLAGS)
GCFLAGS_DEBUG   := $(GCFLAGS) -N -l

# You may need/want to set this to "bundle exec grpc_tools_ruby_protoc"
GRPC_TOOLS_RUBY_PROTOC := grpc_tools_ruby_protoc

default:  release
release:  $(PROTO_GO) $(RELEASE_BUILDS)
debug:    $(PROTO_GO) $(DEBUG_BUILDS)
all:      release debug $(PROTO_RB) $(RPC_RB)
proto:    $(PROTO_GO) $(PROTO_RB) $(RPC_RB)

build/debug/%: $(GOFILES) $(PROTO_GO)
	@mkdir -p "$(@D)"
	@echo "     \x1b[1;34mgo build \x1b[0;1m(debug)\x1b[0m  $@"
	@$(GO) build -trimpath -i -o "$@" -gcflags "$(GCFLAGS_DEBUG)" "$(MODULE)/cmd/$(@F)"

build/release/%: $(GOFILES) $(PROTO_GO)
	@mkdir -p "$(@D)"
	@echo "   \x1b[1;34mgo build \x1b[0;1m(release)\x1b[0m  $@"
	@$(GO) build -trimpath -i -o "$@" -gcflags "$(GCFLAGS_RELEASE)" "$(MODULE)/cmd/$(@F)"

pkg/proto/%/proto.pb.go: proto/%.proto
	@mkdir -p "$(@D)"
	@echo "          \x1b[1;34mprotoc \x1b[0;1m(go)\x1b[0m  $@"
	@$(PROTOC) --go_out=plugins=grpc:. "--proto_path=$(*D)" "$<"
	@mv "$(@D)/$(patsubst %.proto,%,$(*F)).pb.go" "$(@D)/proto.pb.go"

test/lib/protocol/%_pb.rb: proto/%.proto
	@mkdir -p "$(@D)"
	@echo "          \x1b[1;34mprotoc \x1b[0;1m(rb)\x1b[0m  $@"
	@grpc_tools_ruby_protoc -I ./proto --ruby_out=test/lib/protocol "$<"

test/lib/protocol/%_services_pb.rb: proto/%.proto
	@mkdir -p "$(@D)"
	@echo "          \x1b[1;34mprotoc \x1b[0;1m(rb)\x1b[0m  $@"
	@grpc_tools_ruby_protoc -I ./proto --grpc_out=test/lib/protocol "$<"

test: debug $(PROTO_RB) $(RPC_RB)
	ruby $(foreach file,$(TEST_FILES),-r ./$(file)) -e ''

clean:
	@echo "                   \x1b[1;31mrm\x1b[0m  $(CMDS)"
	@rm -f $(CMDS)
	@echo "                   \x1b[1;31mrm\x1b[0m  build"
	@rm -rf build

clean-proto:
	@echo "                   \x1b[1;31mrm\x1b[0m  $(PROTO_GO)"
	@rm -f $(PROTO_GO)
	@echo "                   \x1b[1;31mrm\x1b[0m  $(PROTO_RB)"
	@rm -f $(PROTO_RB)
	@echo "                   \x1b[1;31mrm\x1b[0m  $(RPC_RB)"
	@rm -f $(RPC_RB)

.PHONY: all default release generate clean test proto clean-proto
