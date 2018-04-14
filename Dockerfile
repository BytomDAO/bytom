# build the image
# docker build -t bytom .

# Build Bytom in a stock Go builder container
FROM golang:1.9-alpine as builder

RUN apk add --no-cache make git

ADD . /go/src/github.com/bytom
RUN cd /go/src/github.com/bytom && make bytomd && make bytomcli

# Pull Bytom into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go/src/github.com/bytom/cmd/bytomd/bytomd /usr/local/bin/
COPY --from=builder /go/src/github.com/bytom/cmd/bytomcli/bytomcli /usr/local/bin/

EXPOSE 1999 46656 46657 9888
# CMD ["bytomd", "node", "--help"]

## sample commands
# docker run -it -p 9888:9888 -v ~/.bytomd:/.bytomd bytom:latest
# bytomd init --chain_id testnet
# bytomd node --web.closed --auth.disable
# exit

# docker run -d -p 9888:9888 -v ~/.bytomd:/.bytomd bytom:latest bytomd node --web.closed --auth.disable
# docker container ls
# docker stop <containerId>