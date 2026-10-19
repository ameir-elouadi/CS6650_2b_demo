# Build
FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum* ./
COPY *.go ./
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o product-service .

# Run
FROM --platform=linux/amd64 alpine:3.20
COPY --from=build /app/product-service /product-service
EXPOSE 8080
CMD ["/product-service"]
