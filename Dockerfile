###
# Step 1 - Compile
###
FROM golang:1.14-alpine AS builder

ARG component=${component:-key-retrieval}
ARG branch
ARG revision

ENV GO111MODULE=on
ENV USER=covidshield
ENV UID=10001
ENV GOLDFLAGS="-X github.com/CovidShield/server/pkg/server.branch=${branch} -X github.com/CovidShield/server/pkg/server.revision=${revision}"

WORKDIR /go/src/github.com/CovidShield/server

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="${GOLDFLAGS}" -o server ./cmd/${component}

###
# Step 2 - Build
###
FROM scratch

ENV USER=covidshield

WORKDIR /usr/local/bin

# Import the user and group files from step 1
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder --chown=${USER}:${USER} /go/src/github.com/CovidShield/server/config.yaml /usr/local/bin/config.yaml
COPY --from=builder --chown=${USER}:${USER} /go/src/github.com/CovidShield/server/server /usr/local/bin/server

USER ${USER}:${USER}

# hadolint ignore=DL3025
ENTRYPOINT ["server", "--config_file_path", "./"]
