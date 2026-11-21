FROM golang:1.23-alpine AS build

RUN apk add --no-cache ca-certificates curl && \
    curl -fsSk \
      https://gu-st.ru/content/lending/russian_trusted_root_ca_pem.crt \
      -o /usr/local/share/ca-certificates/russian_trusted_root_ca.crt && \
    curl -fsSk \
      https://gu-st.ru/content/lending/russian_trusted_sub_ca_pem.crt \
      -o /usr/local/share/ca-certificates/russian_trusted_sub_ca.crt && \
    update-ca-certificates

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

RUN go test ./...
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/api \
    ./cmd/api

FROM scratch

COPY --from=build /out/api /api
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

EXPOSE 8080

USER 65532:65532

ENTRYPOINT ["/api"]
