ARG ONOS_KPIMON_BASE_VERSION=latest

FROM onosproject/onos-kpimon-base:$ONOS_KPIMON_BASE_VERSION as base

FROM alpine:3.11
RUN apk add libc6-compat

USER nobody

COPY --from=base /go/src/github.com/onosproject/onos-kpimon/build/_output/onos-kpimon /usr/local/bin/onos-kpimon

ENTRYPOINT ["onos-kpimon"]
