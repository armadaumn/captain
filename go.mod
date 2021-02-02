module github.com/armadanet/captain

go 1.13

replace github.com/armadanet/spinner => ../spinner

replace github.com/armadanet/captain/dockercntrl => ./dockercntrl

require (
	github.com/armadanet/captain/dockercntrl v0.0.0-20200130235059-2b593e57fe6c
	github.com/armadanet/spinner v0.0.0-00010101000000-000000000000
	github.com/mmcloughlin/geohash v0.10.0
	google.golang.org/grpc v1.34.1
)
