# 本番用 API コンテナ（マイグレーションは起動時 main が実行）
FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/chat-app .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/chat-app .
ENV PORT=8080
EXPOSE 8080
CMD ["./chat-app"]
