# 多阶段构建：Go 网关（健康检查等北向 HTTP）
FROM golang:1.21-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/gateway ./cmd/gateway

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/gateway /app/gateway
# 默认配置打入镜像（compose 可通过卷覆盖）
COPY config.yaml /app/config.yaml
EXPOSE 8080
USER nobody:nobody
ENTRYPOINT ["/app/gateway"]
CMD ["-config", "/app/config.yaml"]
