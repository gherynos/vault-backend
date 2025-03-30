# Build the application
FROM golang:1.23.5 as build

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build -o /go/bin/app

# Base image
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /go/bin/app /

EXPOSE 8080

CMD ["/app"]
