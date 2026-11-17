FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -o /redirect .

FROM alpine:3.20
COPY --from=build /redirect /redirect
EXPOSE 8080
ENTRYPOINT ["/redirect"]
