all:
	GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o petcam
	scp petcam mk@rpi:~/
	rm petcam

