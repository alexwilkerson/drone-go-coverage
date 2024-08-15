FROM --platform=linux/amd64 golang:1.23-alpine3.19
ADD drone-go-coverage /bin/
RUN chmod +x /bin/drone-go-coverage
ENTRYPOINT ["/bin/drone-go-coverage"]
