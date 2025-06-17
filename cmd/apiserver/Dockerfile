# 构建阶段
FROM golang:1.24-alpine AS builder

WORKDIR /app

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
# if you are in China, you can uncomment the following line
# RUN go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
WORKDIR /app/cmd/apiserver
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /apiserver .

# 运行阶段
FROM alpine:latest

# 安装ca-certificates用于HTTPS请求
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /apiserver .

# 暴露端口
EXPOSE 8080

# 运行应用
CMD ["./apiserver"]
