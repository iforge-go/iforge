# ---------- 构建阶段 ----------
FROM golang:1.26-alpine AS builder

# 国内构建走 goproxy.cn 镜像，避免 proxy.golang.org 超时
# 海外环境可改为 https://proxy.golang.org,direct
ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /build

# 先复制依赖文件，利用 Docker 层缓存（依赖不变时跳过 go mod download）
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 静态编译
# - CGO_ENABLED=0：项目使用 glebarez/sqlite 纯 Go 驱动，无需 CGO
# - -ldflags="-s -w": 去除调试信息，缩小二进制体积
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /iforge ./cmd/server

# ---------- 运行阶段 ----------
FROM alpine:3.20

# 国内构建替换 apk 源为阿里云镜像，避免 dl-cdn.alpinelinux.org 访问失败
# 海外环境可删除此 sed（默认源全球可访问）
# 运行时依赖：
# - git: 仓库管理（go-git 大部分操作不需要，但保留以兼容外部命令）
# - ca-certificates: HTTPS 请求
# - openssh: SSH Git 服务（git@host:repo 形式的克隆/推送）
# - tzdata: 时区数据
RUN sed -i 's|dl-cdn.alpinelinux.org|mirrors.aliyun.com|g' /etc/apk/repositories && \
    apk add --no-cache git ca-certificates openssh tzdata

WORKDIR /app

COPY --from=builder /iforge /app/iforge

# 数据目录（IFORGE_HOME）：数据库、日志、Git 仓库都存这里
# 通过卷挂载持久化，容器重建后数据不丢失
ENV IFORGE_HOME=/app/data
RUN mkdir -p /app/data

# HTTP（8081）+ SSH（2022）
EXPOSE 8081 2022

ENTRYPOINT ["/app/iforge"]
