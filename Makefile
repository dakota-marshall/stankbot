build:
	# Statically compile for portability
	CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' .
	docker build . -t stankbot
run:
	docker run --rm -it --env-file=.env stankbot
clean:
	rm -rf stankbot
