FROM golang:1.27-alpine AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOFLAGS=-trimpath go build -ldflags "-s -w -X main.version=${VERSION}" -o /out/rosiebot ./cmd/rosiebot

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/rosiebot /rosiebot
USER nonroot:nonroot
ENTRYPOINT ["/rosiebot"]
