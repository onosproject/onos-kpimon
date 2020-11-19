module github.com/onosproject/onos-kpimon

go 1.14

require (
	github.com/onosproject/onos-e2t v0.6.7-0.20201112232226-f90757e4b4c0
	github.com/onosproject/onos-lib-go v0.6.25
	github.com/onosproject/onos-ric-sdk-go v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.33.2
)
replace github.com/onosproject/onos-ric-sdk-go => ../onos-ric-sdk-go
