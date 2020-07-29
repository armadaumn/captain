FROM codyperakslis/dev-go:1.14.4-v1
COPY captain/go.mod captain/go.sum captain/
ADD spinner/ /home/dev/spinner/
ADD captain/dockercntrl/ /home/dev/captain/dockercntrl/
RUN cd captain && go mod download 
ADD captain/ /home/dev/captain/
WORKDIR /home/dev/captain
RUN go build -o main cmd/captain.go
ENTRYPOINT [ "./main" ]
