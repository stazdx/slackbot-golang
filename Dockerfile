FROM golang:1.17.6-alpine as Build

COPY . . 

RUN GOPATH= go build -o /main main.go 

FROM scratch

COPY --from=Build main main

ENTRYPOINT [ "/main" ]