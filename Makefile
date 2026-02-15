all:
	cp scripts/stream_pi.sh scripts/stream.sh
	GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o petcam
	ssh mk@rpi mkdir -p ~/petcam/scripts
	scp petcam mk@rpi:~/petcam/
	scp scripts/stream.sh mk@rpi:~/petcam/scripts/
	rm petcam
	rm -f scripts/stream.sh

dev:
	cp scripts/stream_dev.sh scripts/stream.sh
	go build -o petcam

run: dev
	./petcam

clean:
	rm -f petcam scripts/stream.sh

test:
	go test -race ./...
