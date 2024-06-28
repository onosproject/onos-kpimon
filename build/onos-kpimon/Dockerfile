# SPDX-FileCopyrightText: 2019-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

ARG ONOS_BUILD_VERSION=undefined

FROM onosproject/golang-build:v1.3.0 as build

ENV GO111MODULE=on
ARG ONOS_MAKE_TARGET=build

COPY Makefile go.mod go.sum /go/src/github.com/onosproject/onos-kpimon/
COPY cmd/ /go/src/github.com/onosproject/onos-kpimon/cmd/
COPY pkg/ /go/src/github.com/onosproject/onos-kpimon/pkg/
COPY vendor/ /go/src/github.com/onosproject/onos-kpimon/vendor/
COPY api/ /go/src/github.com/onosproject/onos-kpimon/api
COPY build/ /go/src/github.com/onosproject/onos-kpimon/build/

RUN cd /go/src/github.com/onosproject/onos-kpimon && GOFLAGS=-mod=vendor make ${ONOS_MAKE_TARGET}

FROM frolvlad/alpine-glibc:alpine-3.12

USER nobody

COPY --from=build /go/src/github.com/onosproject/onos-kpimon/build/_output/onos-kpimon /usr/local/bin/onos-kpimon

ENTRYPOINT ["onos-kpimon"]
