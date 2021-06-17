[![Build Status](https://travis-ci.org/open-nebula/captain.svg?branch=master)](https://travis-ci.org/open-nebula/captain)

# Captain
Captain is a computation worker node in project [Armada](https://armadanet.github.io/). It runs task container in Docker
deployed by application deployers.

## What is this?
When a machine wants to join Armada system and contributes its computation power, it needs to run this captain module to
manage its local task containers and connect to other modules in Armada. It includes receive application deployment requests, 
set up task Docker containers, container configuration, and internal network connections. Note that the captain module 
will access to the local Docker engine to perform container-related operations. The resource contribued to Armada is limited 
to the resources allocated to Docker engine.

## Quick Start
**Prerequisites**: Docker

To download the spinner image run: \
```docker pull armadaumn/captain:latest``` \
To start the spinner just run: \
```docker run -it --rm -v /var/run/docker.sock:/var/run/docker.sock armadaumn/captain:latest $SERVER_TYPE $LOC $TAG $SPINNER_URL``` \
Arguments:
* $SERVER_TYPE: indicate the type of the current machine, "Sserver" for a local server, "volunteer" for a personal machine.
* $LOC: The location of the current machine. The captain module can locate the machine automatically, so this field is just a placeholder for now.
* $TAG: Any tag for the machine.
* $SPINNER_URL: the ip address of the [spinner](https://github.com/armadanet/spinner) 

## Build from the source
**Prerequisites**: Go environment, Docker

Build and run spinner:
```
git clone https://github.com/armadanet/captain.git
cd captain/build
make build
make run
```
