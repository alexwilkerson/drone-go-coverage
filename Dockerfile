FROM golang:1.23.1-alpine3.20
ADD drone-go-coverage /bin/
RUN chmod +x /bin/drone-go-coverage
ENTRYPOINT ["/bin/drone-go-coverage"]
