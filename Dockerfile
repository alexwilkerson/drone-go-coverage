FROM --platform=linux/amd64 golang:1.21-alpine3.18
ADD drone-go-coverage /bin/
RUN chmod +x /bin/drone-go-coverage
ENTRYPOINT ["/bin/drone-go-coverage"]
