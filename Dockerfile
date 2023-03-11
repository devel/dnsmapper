FROM golang:1.20.2-alpine3.17 AS build
RUN apk add git
WORKDIR /go/src/github.com/devel/dnsmapper
ADD . /go/src/github.com/devel/dnsmapper
RUN go install github.com/devel/dnsmapper/mist
RUN go install github.com/devel/dnsmapper/store

FROM alpine:3.17.2
RUN apk --no-cache add ca-certificates

RUN addgroup dnsmapper && adduser -D -G dnsmapper dnsmapper

#RUN chown dnsmapper:dnsmapper /dnsmapper/tmp ?

WORKDIR /dnsmapper/

COPY --from=build /go/bin/mist  /dnsmapper/
COPY --from=build /go/bin/store /dnsmapper/

ADD run-store /dnsmapper/

# COPY --from=build /go/src/git.develooper.com/project/templates /project/templates/
# COPY --from=build /go/src/git.develooper.com/project/static /project/static/
# COPY --from=build /go/src/git.develooper.com/project/config.yaml.sample /etc/project/config.yaml

USER dnsmapper
