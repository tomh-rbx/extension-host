# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:1.20-bullseye AS build

ARG NAME
ARG VERSION
ARG REVISION
ARG ADDITIONAL_BUILD_PARAMS

WORKDIR /app


RUN apt-get -qq update \
    && apt-get -qq install -y --no-install-recommends build-essential libcap2-bin
COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build \
    -ldflags="\
    -X 'github.com/steadybit/extension-kit/extbuild.ExtensionName=${NAME}' \
    -X 'github.com/steadybit/extension-kit/extbuild.Version=${VERSION}' \
    -X 'github.com/steadybit/extension-kit/extbuild.Revision=${REVISION}'" \
    -o ./extension \
    ${ADDITIONAL_BUILD_PARAMS}

##
## Runtime
##
FROM debian:bullseye-slim

ARG USERNAME=steadybit
ARG USER_UID=10000
ARG USER_GID=$USER_UID

RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME

RUN apt-get -qq update \
    && apt-get -qq install -y --no-install-recommends procps stress-ng iptables iproute2 dnsutils libcap2-bin \
    && apt-get -y autoremove \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /opt/steadybit/extension

ADD ./check-tools.sh /opt/steadybit/extension/check/
RUN /opt/steadybit/extension/check/check-tools.sh

COPY --from=build /app/extension /opt/steadybit/extension/extension
RUN chown -R $USERNAME:$USERNAME /opt/steadybit/extension
RUN setcap "cap_sys_boot,cap_sys_time,cap_setuid,cap_setgid,cap_net_raw,cap_net_admin,cap_sys_admin,cap_dac_override+eip" /opt/steadybit/extension/extension
USER $USERNAME



WORKDIR /opt/steadybit/extension


EXPOSE 8085 8081

ENTRYPOINT ["/opt/steadybit/extension/extension"]
