FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app

FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache python3 py3-pip

COPY requirements.txt .
RUN pip3 install --no-cache-dir -r requirements.txt

COPY --from=builder /app/app .
COPY --from=builder /app/etc ./etc
COPY --from=builder /app/internal/crawler ./internal/crawler

EXPOSE 8888

CMD ["./app", "-f", "./etc/local/api.yaml"]
