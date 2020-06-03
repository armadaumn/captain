module github.com/armadanet/captain

go 1.13

replace github.com/armadanet/captain/dockercntrl => ./dockercntrl

require (
	github.com/armadanet/captain/dockercntrl v0.0.0-20200130235059-2b593e57fe6c
	github.com/armadanet/comms v0.0.0-20200528090635-3089ce4375d7
	github.com/armadanet/spinner/spinresp v0.0.0-20200130235212-5ec32922cd99
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.4
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	golang.org/x/net v0.0.0-20200114155413-6afb5195e5aa
)
