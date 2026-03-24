# 编译文件
# Docker 分离构建环境和运行环境是为了减少镜像体积与攻击面
# 本质是在宿主机上运行的一个被隔离的普通进程。
FROM golang:1.26.0-alpine AS builder

WORKDIR /app

# Docker 的缓存是按层来判断的，因此我们尽量让每次被改变的缓存少，防止重复加载
# 所以 Docker 本质上就是一个多层文件复用系统，每层叠加的只有变化
# RUN 是构建阶段命令，而 CMD 是运行阶段默认命令
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://goproxy.cn,direct && \
    go env -w GOSUMDB=sum.golang.google.cn && \
    go mod download 
COPY . .

# 运行编译出来的文件
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app

FROM alpine:latest

WORKDIR /app

# 安装 python3 和 pip
RUN apk add --no-cache python3 py3-pip

# 复制依赖文件
COPY requirements.txt .

# 安装依赖
RUN pip3 install --no-cache-dir -r requirements.txt

# 复制 Go 编译产物
COPY --from=builder /app/app .

# 复制爬虫脚本
COPY crawler_cli.py .

EXPOSE 8888

CMD ["./app", "-f", "./etc/local/api.yaml"]
