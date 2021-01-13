IMAGE := nebula-captain
REPO := codyperakslis/captain
URL := wss://69de8bbe.ngrok.io/join
NAME := primary
NETWORK := spinner-local-network

# Dynamic Port Exposure
# https://stackoverflow.com/questions/19897743/exposing-a-port-on-a-live-docker-container

# Or using host network (doesn't work for Mac because host is VM)

build:
	cd ../.. && docker build -t $(IMAGE) -f captain/build/Dockerfile .

run:
	docker run -it --rm -v /var/run/docker.sock:/var/run/docker.sock --net $(NETWORK) $(IMAGE) 

