# ---- Stage 1: build ----
FROM golang:1.27-alpine AS builder
WORKDIR /app

# Copy only the dependency files first, so Docker can cache this step.
COPY go.mod go.sum ./
RUN go mod download

# Now copy the rest of the code and compile it.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/order-service ./cmd/order-service

# ---- Stage 2: run ----
FROM alpine:3.20
WORKDIR /app
COPY --from=builder /out/order-service .
EXPOSE 8080
CMD ["./order-service"]