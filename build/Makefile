IMAGE := nebula-captain
REPO := codyperakslis/captain
#URL := wss://69de8bbe.ngrok.io/join
URL := http://18.221.23.45:9898/newCaptain
NAME := primary

# Dynamic Port Exposure
# https://stackoverflow.com/questions/19897743/exposing-a-port-on-a-live-docker-container

# Or using host network (doesn't work for Mac because host is VM)

build:
	cd .. && docker build -t $(IMAGE) -f build/Dockerfile .

run: build
	docker run -it --name $(NAME) -v /var/run/docker.sock:/var/run/docker.sock $(IMAGE) $(URL) $(NAME)

clean:
	docker rm $(NAME)
	docker rmi $(IMAGE)
	docker swarm leave -f
