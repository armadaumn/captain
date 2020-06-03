IMAGE := nebula-captain
REPO := codyperakslis/captain
#URL := wss://69de8bbe.ngrok.io/join
URL := http://3.23.114.100:9898/newCaptain
NAME := captain1
# selfSpin and spinner_name for selfspin
SELFSPIN := false
SPINNER_NAME := spinner1
BEACON_QUERY := http://3.23.114.100:9898/newSpinner


build:
	cd .. && docker build -t $(IMAGE) -f build/Dockerfile .

run: build
	docker run -it --name $(NAME) -e SELFSPIN=$(SELFSPIN) -e SPINNER_NAME=$(SPINNER_NAME) -e BEACON_QUERY=$(BEACON_QUERY) -v /var/run/docker.sock:/var/run/docker.sock $(IMAGE) $(URL) $(NAME)

clean:
	docker rm $(NAME)
	docker rmi $(IMAGE)
	docker swarm leave -f
	#docker stop armada-storage
	docker stop $(SPINNER_NAME) armada-storage
	#docker rm armada-storage
	docker rm $(SPINNER_NAME) armada-storage
	#docker rmi geoffreyhl/armada-cargo
	docker rmi geoffreyhl/spinner geoffreyhl/armada-cargo
	docker network rm armada_bridge
	docker volume rm cargo
