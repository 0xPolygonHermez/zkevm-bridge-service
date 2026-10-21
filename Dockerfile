# CONTAINER FOR BUILDING BINARY
FROM golang:1.26.10 AS build

ENV CGO_ENABLED=0
# INSTALL DEPENDENCIES
RUN go install github.com/gobuffalo/packr/v2/packr2@v2.8.3
COPY go.mod go.sum /src/
RUN cd /src && go mod download

# BUILD BINARY
COPY . /src
RUN cd /src/db && packr2
RUN cd /src && make build

# CONTAINER FOR RUNNING BINARY
FROM postgres:latest
RUN apt-get update
RUN apt-get install ca-certificates -y
COPY --from=build /src/dist/zkevm-bridge /app/zkevm-bridge
COPY --from=build /src/dist/test-deploy-tool /app/test-deploy-tool
COPY --from=build /src/dist/zkevm-autoclaimer /app/zkevm-autoclaimer
COPY --from=build /src/test/vectors /app/test/vectors
EXPOSE 8080
EXPOSE 9090
CMD ["/bin/sh", "-c", "/app/zkevm-bridge run"]
