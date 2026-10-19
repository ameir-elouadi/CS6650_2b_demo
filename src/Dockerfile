# Build
FROM golang:1.25-alpine
WORKDIR /app
COPY go.mod go.sum* ./
COPY *.go ./
RUN go mod tidy && go build -o product-service .

# Run
EXPOSE 8080
CMD ["./product-service"]
