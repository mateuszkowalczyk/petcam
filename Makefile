REMOTE_USER ?= mk
REMOTE_HOST ?= rpi
LED_NAME ?= ACT
REMOTE = $(REMOTE_USER)@$(REMOTE_HOST)

.PHONY: all dev run clean test

all: build-arm deploy clean

build-arm:
	cp scripts/stream_pi.sh scripts/stream.sh
	GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o petcam

deploy:
	sed 's/@REMOTE_USER@/$(REMOTE_USER)/g; s/@LED_NAME@/$(LED_NAME)/g' scripts/petcam.service.template > scripts/petcam.service
	sed 's/@LED_NAME@/$(LED_NAME)/g' scripts/install.sh.template > scripts/install.sh
	chmod +x scripts/install.sh
	ssh $(REMOTE) mkdir -p ~/petcam/scripts
	scp petcam $(REMOTE):~/petcam/
	scp scripts/stream.sh $(REMOTE):~/petcam/scripts/
	scp scripts/petcam.service $(REMOTE):~/petcam/scripts/
	scp scripts/install.sh $(REMOTE):~/petcam/scripts/
	scp scripts/uninstall.sh $(REMOTE):~/petcam/scripts/

dev:
	cp scripts/stream_dev.sh scripts/stream.sh
	go build -o petcam

run: dev
	./petcam

clean:
	rm -f petcam scripts/stream.sh scripts/petcam.service scripts/install.sh

test:
	go test -race ./...
