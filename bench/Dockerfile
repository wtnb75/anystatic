FROM golang:1-alpine AS build
ARG BRANCH=latest
RUN go install github.com/wtnb75/anystatic/cmd/anystatic@${BRANCH}
RUN go install github.com/wtnb75/anystatic/cmd/compr-recursive@${BRANCH}

FROM alpine:3
COPY --from=build /go/bin/anystatic /
COPY --from=build /go/bin/compr-recursive /
RUN apk add wrk brotli zstd gzip python3 curl
ADD makedata.sh /
ADD mark.sh /
