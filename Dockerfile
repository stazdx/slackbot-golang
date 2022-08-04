FROM golang:1.17.6-alpine as Build

COPY . . 
COPY .env /.env

RUN GOPATH= go build -o /main main.go 

FROM alpine

USER root

COPY --from=Build main main
COPY --from=Build .env .env

ENTRYPOINT [ "/main" ]