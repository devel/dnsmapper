FROM golang:1.21.3-alpine3.18 AS build
RUN apk --no-cache add git
WORKDIR /go/src/github.com/devel/dnsmapper
ADD . /go/src/github.com/devel/dnsmapper
RUN git config --global --add safe.directory /go/src/github.com/devel/dnsmapper
RUN go get -v ./...
RUN go install .
RUN go install github.com/devel/dnsmapper/mist
RUN go install github.com/devel/dnsmapper/store

FROM alpine:3.18.4
RUN apk --no-cache add ca-certificates

RUN addgroup dnsmapper && adduser -D -G dnsmapper dnsmapper

#RUN chown dnsmapper:dnsmapper /dnsmapper/tmp ?

WORKDIR /dnsmapper/

COPY --from=build /go/bin/dnsmapper /dnsmapper/
COPY --from=build /go/bin/mist  /dnsmapper/
COPY --from=build /go/bin/store /dnsmapper/

ADD run-store /dnsmapper/

# COPY --from=build /go/src/git.develooper.com/project/templates /project/templates/
# COPY --from=build /go/src/git.develooper.com/project/static /project/static/
# COPY --from=build /go/src/git.develooper.com/project/config.yaml.sample /etc/project/config.yaml

USER dnsmapper
