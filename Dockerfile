# CONTAINER FOR BUILDING BINARY
FROM golang:1.17 AS build

ENV CGO_ENABLED=1
ARG PRIVATE_TOKEN
# INSTALL DEPENDENCIES
RUN go install github.com/gobuffalo/packr/v2/packr2@v2.8.3
COPY go.mod go.sum /src/
RUN git config --global url."https://${PRIVATE_TOKEN}:x-oauth-basic@github.com/".insteadOf "https://github.com/"
RUN cd /src && go mod download

# BUILD BINARY
COPY . /src
RUN cd /src/db && packr2
RUN cd /src && make build

# CONTAINER FOR RUNNING BINARY
FROM golang:1.17
WORKDIR /app
COPY --from=build /src/dist/hezbridge /app/hezbridge
COPY --from=build /src/test/vectors /app/test/vectors
EXPOSE 8124
EXPOSE 8080
CMD ["./hezbridge", "run"]
