# builder stage
FROM golang:1.14-alpine AS builder
ARG component=${component:-key-retrieval}
ENV GO111MODULE=on
WORKDIR /go/src/github.com/CovidShield/backend
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o ${component} ./cmd/${component}

# target stage
FROM alpine:3
ARG component=${component:-key-retrieval}
ENV component=${component}
ARG APP_UID=${APP_UID:-2000}
ARG APP_GID=${APP_GID:-2000}
RUN addgroup -g ${APP_GID} -S ${component} && \
    adduser -u ${APP_UID} -S ${component} -G ${component}
COPY --from=builder --chown=${APP_UID}:${APP_GID} /go/src/github.com/CovidShield/backend/${component} /usr/local/bin/${component}

USER ${APP_UID}:${APP_GID}

# hadolint ignore=DL3025
ENTRYPOINT "/usr/local/bin/${component}"
