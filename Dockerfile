###
# Step 1 - Compile
###
FROM golang:1.15.6-alpine3.12 AS builder

ARG component=${component:-key-retrieval}
ARG branch
ARG revision

ENV GO111MODULE=on
ENV USER=covidshield
ENV UID=10001
ENV GOLDFLAGS="-X github.com/cds-snc/covid-alert-server/pkg/server.branch=${branch} -X github.com/cds-snc/covid-alert-server/pkg/server.revision=${revision}"

WORKDIR /go/src/github.com/cds-snc/covid-alert-server

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

RUN mkdir /etc/aws-certs
RUN wget -P /etc/aws-certs https://s3.amazonaws.com/rds-downloads/rds-ca-2019-root.pem

###
# Step 2 - Build
###
FROM scratch

ENV USER=covidshield

WORKDIR /usr/local/bin

# Import the user and group files from step 1
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /etc/aws-certs /etc/aws-certs
COPY --from=builder --chown=${USER}:${USER} /go/src/github.com/cds-snc/covid-alert-server/config.yaml /usr/local/bin/config.yaml
COPY --from=builder --chown=${USER}:${USER} /go/src/github.com/cds-snc/covid-alert-server/server /usr/local/bin/server

USER ${USER}:${USER}

# hadolint ignore=DL3025
ENTRYPOINT ["server", "--config_file_path", "./"]
