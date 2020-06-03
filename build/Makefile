IMAGE := nebula-captain
REPO := codyperakslis/captain
#URL := wss://69de8bbe.ngrok.io/join
URL := http://3.23.114.100:9898/newCaptain
NAME := primary
# selfSpin and spinner_name for selfspin
SELFSPIN := false
SPINNER_NAME := spinner1


build:
	cd .. && docker build -t $(IMAGE) -f build/Dockerfile .

run: build
	docker run -it --name $(NAME) -e SELFSPIN=$(SELFSPIN) -e SPINNER_NAME=$(SPINNER_NAME) -v /var/run/docker.sock:/var/run/docker.sock $(IMAGE) $(URL) $(NAME)

clean:
	docker rm $(NAME)
	docker rmi $(IMAGE)
	docker swarm leave -f
