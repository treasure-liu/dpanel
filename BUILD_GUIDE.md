# DPanel 构建与使用指南

本文档提供了从源码构建 DPanel 镜像并使用容器告警功能的完整指南。

## 目录

- [源码构建指南(推荐使用Dockerfile.multi进行整体构建)](#源码构建指南)
  - [Go环境安装](#go环境安装)
  - [安装编译依赖](#安装编译依赖)
- [Docker多阶段构建(推荐)](#docker多阶段构建推荐)
- [快速启动](#快速启动) 
- [容器告警配置](#容器告警配置)
- [钉钉告警配置](#钉钉告警配置)
- [邮件告警配置](#邮件告警配置)
- [飞书告警配置](#飞书告警配置)
- [企业微信告警配置](#企业微信告警配置)
- [常见问题](#常见问题)

## 源码构建指南

### 前提条件

- Go 1.16或更高版本
- Docker
- Git
- CGO支持（编译SQLite必需）

### Go环境安装

如果您的系统中尚未安装Go环境，请按照以下步骤进行安装：

#### Linux安装Go

```bash
# 下载Go安装包（以amd64架构为例，版本号可替换为最新版）
wget https://golang.org/dl/go1.18.linux-amd64.tar.gz

# 解压到/usr/local目录
sudo tar -C /usr/local -xzf go1.18.linux-amd64.tar.gz

# 设置环境变量
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
echo 'export GOPATH=$HOME/go' >> ~/.profile
source ~/.profile

# 验证安装
go version
```

#### Windows安装Go

1. 从[Go官方网站](https://golang.org/dl/)下载Windows安装包
2. 运行下载的MSI安装文件，按照安装向导完成安装
3. 安装程序会自动将Go添加到PATH环境变量
4. 打开命令提示符或PowerShell，验证安装：
   ```
   go version
   ```

#### macOS安装Go

```bash
# 使用Homebrew安装
brew install go

# 或者下载安装包手动安装
# 从 https://golang.org/dl/ 下载macOS安装包
# 双击下载的.pkg文件，按照安装向导完成安装

# 验证安装
go version
```

#### 设置GOPROXY (推荐，加速依赖下载)

对于中国大陆用户，建议设置GOPROXY以加速依赖下载：

```bash
# 设置为国内镜像
go env -w GOPROXY=https://goproxy.cn,direct

# 或者使用阿里云镜像
# go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
```

### 安装编译依赖

DPanel使用SQLite数据库，需要CGO支持，请确保安装必要的系统依赖：

#### Linux（Ubuntu/Debian）
```bash
apt-get update
apt-get install -y build-essential libsqlite3-dev
```

#### Linux（CentOS/RHEL）
```bash
yum groupinstall -y "Development Tools"
yum install -y sqlite-devel
```

#### macOS
```bash
brew install sqlite
```

#### Windows
需要安装MinGW或MSYS2环境和SQLite开发库。

### 1. 获取源码

```bash
git clone https://gitee.com/treasure_liu/dpanel.git
# Github原地址
git clone https://github.com/donknap/dpanel.git
cd dpanel
```

### 2. 使用Makefile构建（推荐）

项目提供了Makefile来简化构建过程。**注意：所有构建命令必须启用CGO_ENABLED=1，否则SQLite无法正常工作**。

```bash
# 构建所有平台版本
make all

# 仅构建当前平台二进制文件
make build

# 构建Linux amd64架构
make amd64

# 构建Linux arm64架构
make arm64

# 构建Linux armv7架构
make armv7
```

### 3. 手动构建

如果不使用Makefile，需要手动执行以下步骤：

#### a. 编译二进制文件

**重要：必须设置CGO_ENABLED=1，确保SQLite数据库可以正常工作**

```bash
# 创建输出目录
mkdir -p runtime

# 编译（根据您的目标平台选择对应的GOOS和GOARCH）
# Linux AMD64:
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o runtime/dpanel-musl-amd64 main.go

# Linux ARM64:
CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o runtime/dpanel-musl-arm64 main.go

# 拷贝配置文件
cp config.yaml runtime/config.yaml
```

#### b. 跨平台编译（使用musl-cross）

如果需要在macOS或其他平台上交叉编译Linux二进制文件，需要安装相应的交叉编译工具链：

```bash
# macOS上安装musl交叉编译工具链
brew tap messense/macos-cross-toolchains
brew install x86_64-linux-musl
brew install aarch64-unknown-linux-musl
brew install armv7-unknown-linux-musleabihf

# 然后使用工具链编译
CGO_ENABLED=1 GOARCH=amd64 GOOS=linux CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++ \
go build -ldflags="-w -s" -o runtime/dpanel-musl-amd64 main.go
```

#### c. 构建Docker镜像

标准版（包含Nginx和域名转发功能）：

```bash
docker build --build-arg TARGETARCH=amd64 --build-arg APP_VERSION=latest -t dpanel:latest .
```

精简版（不包含Nginx，只提供API服务）：

```bash
docker build -f Dockerfile-lite --build-arg TARGETARCH=amd64 --build-arg APP_VERSION=latest -t dpanel:lite .
```

## Docker多阶段构建(推荐)

为了避免跨平台编译问题和SQLite依赖问题，强烈推荐使用Docker多阶段构建方式。这种方式可以在同一个环境中完成编译和运行，确保二进制文件正常工作。

### 1. 使用多阶段构建文件

项目提供了两个多阶段构建的Dockerfile:

- **Dockerfile.multi**: 标准版，包含Nginx和完整功能
- **Dockerfile.multi-lite**: 精简版，不包含Nginx，只提供API服务

这两个文件的优点:
- 无需预先编译二进制文件
- 自动启用CGO并安装所需依赖
- 避免架构不匹配问题
- 确保SQLite数据库正常工作

### 2. 构建镜像

```bash
# 构建标准版
docker build -t dpanel:latest -f Dockerfile.multi --build-arg APP_VERSION=1.0.0 .

# 构建精简版
docker build -t dpanel:lite -f Dockerfile.multi-lite --build-arg APP_VERSION=1.0.0 .

# 构建专业版(pe)
docker build -t dpanel-pe:latest -f Dockerfile.multi --build-arg APP_VERSION=1.0.0 --build-arg APP_FAMILY=pe .
```

### 3. 多架构构建

如果需要构建多架构镜像，可以使用`docker buildx`:

```bash
# 创建并使用buildx构建器
docker buildx create --name dpanel-builder --use

# 构建并推送多架构镜像(标准版)
docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7 \
  -t username/dpanel:latest -f Dockerfile.multi \
  --build-arg APP_VERSION=1.0.0 \
  --push .

# 构建并推送多架构镜像(精简版)
docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7 \
  -t username/dpanel:lite -f Dockerfile.multi-lite \
  --build-arg APP_VERSION=1.0.0 \
  --push .
```

## 快速启动

### 使用预构建镜像

如果您不需要从源码构建，可以直接使用官方预构建镜像：

#### 标准版

```bash
docker run -d --name dpanel --restart=always \
 -p 80:80 -p 443:443 -p 8807:8080 -e APP_NAME=dpanel \
 -v /var/run/docker.sock:/var/run/docker.sock -v dpanel:/dpanel \
 dpanel/dpanel:latest 
```

#### 精简版

```bash
docker run -d --name dpanel --restart=always \
 -p 8807:8080 -e APP_NAME=dpanel \
 -v /var/run/docker.sock:/var/run/docker.sock -v dpanel:/dpanel \
 dpanel/dpanel:lite
```

## 容器告警配置

DPanel 支持监控容器状态并发送告警通知。您可以在构建镜像或启动容器时配置以下环境变量：

### 基本配置

```bash
# 启用容器告警（必须设置为true才能启用功能）
-e CONTAINER_ALERT_ENABLED=true \

# 监控指定容器，多个用逗号分隔，留空表示监控所有容器
-e CONTAINER_ALERT_MONITOR="mysql,nginx,app" \

# 排除特定容器，多个用逗号分隔
-e CONTAINER_ALERT_EXCLUDE="temp,test" \

# 告警类型配置
-e CONTAINER_ALERT_ON_STOP=false \    # 容器停止时是否告警
-e CONTAINER_ALERT_ON_DIE=true \      # 容器异常退出时是否告警
-e CONTAINER_ALERT_ON_OOM=true \      # 容器内存溢出(OOM)时是否告警
-e CONTAINER_ALERT_ON_HEALTH=true \   # 容器健康检查失败时是否告警

# 健康检查间隔（秒）
-e CONTAINER_ALERT_INTERVAL=60 \
```

> **注意**: 关于容器错误码的详细解释和排查指南，请参阅 [Docker容器错误码参考](docs/CONTAINER_ERROR_CODES.md)。该文档提供了常见错误码的含义、可能原因和排查方法。

### 使用场景示例

#### 监控所有容器

```bash
docker run -d --name dpanel --restart=always \
 -p 8807:8080 -e APP_NAME=dpanel \
 -e CONTAINER_ALERT_ENABLED=true \
 -v /var/run/docker.sock:/var/run/docker.sock -v dpanel:/dpanel \
 dpanel/dpanel:lite
```

#### 只监控特定容器

```bash
docker run -d --name dpanel --restart=always \
 -p 8807:8080 -e APP_NAME=dpanel \
 -e CONTAINER_ALERT_ENABLED=true \
 -e CONTAINER_ALERT_MONITOR="mysql,nginx,redis" \
 -v /var/run/docker.sock:/var/run/docker.sock -v dpanel:/dpanel \
 dpanel/dpanel:lite
```

#### 监控所有容器但排除某些容器

```bash
docker run -d --name dpanel --restart=always \
 -p 8807:8080 -e APP_NAME=dpanel \
 -e CONTAINER_ALERT_ENABLED=true \
 -e CONTAINER_ALERT_MONITOR="" \
 -e CONTAINER_ALERT_EXCLUDE="temp,test,dev-*" \
 -v /var/run/docker.sock:/var/run/docker.sock -v dpanel:/dpanel \
 dpanel/dpanel:lite
```

## 钉钉告警配置

DPanel支持将容器告警消息推送到钉钉群机器人：

```bash
# 启用钉钉告警
-e DINGTALK_ENABLED=true \

# 设置钉钉机器人Webhook地址
-e DINGTALK_WEBHOOK_URL="https://oapi.dingtalk.com/robot/send?access_token=xxxx" \

# 设置钉钉安全设置的签名密钥（如果机器人设置了签名验证）
-e DINGTALK_SECRET="SEC000xxxxx" \
```

### 钉钉机器人配置步骤

1. 进入钉钉群 -> 群设置 -> 智能群助手 -> 添加机器人 -> 自定义
2. 配置机器人：
   - 名称：随意设置，如"容器告警"
   - 安全设置：推荐选择"加签"（获取SEC密钥）
   - 权限：允许发送消息
3. 创建后获取Webhook地址和签名密钥（SEC开头的字符串）
4. 将获取到的信息填入环境变量

## 邮件告警配置

DPanel支持通过邮件发送容器告警通知，配置方式如下：

```bash
# 启用邮件告警
-e EMAIL_ENABLED=true \

# SMTP服务器设置
-e EMAIL_HOST="smtp.example.com" \
-e EMAIL_PORT="465" \
-e EMAIL_USERNAME="user@example.com" \
-e EMAIL_PASSWORD="password" \

# 发件人和收件人
-e EMAIL_FROM="user@example.com" \
-e EMAIL_TO="recipient1@example.com,recipient2@example.com" \

# 是否使用SSL连接，一般465端口需要设置为true，25端口设置为false
-e EMAIL_USE_SSL=true \

```

### 常见邮件服务器配置

#### Gmail

```bash
-e EMAIL_HOST="smtp.gmail.com" \
-e EMAIL_PORT="465" \
-e EMAIL_USERNAME="your-email@gmail.com" \
-e EMAIL_PASSWORD="your-app-password" \
-e EMAIL_USE_SSL=true
```

注意：Gmail 需要使用应用专用密码，请参考 [Google 应用专用密码](https://support.google.com/accounts/answer/185833)。

#### QQ邮箱

```bash
-e EMAIL_HOST="smtp.qq.com" \
-e EMAIL_PORT="465" \
-e EMAIL_USERNAME="your-qq-number@qq.com" \
-e EMAIL_PASSWORD="your-authorization-code" \
-e EMAIL_USE_SSL=true
```

注意：QQ邮箱需要使用授权码，不是QQ密码，请在QQ邮箱设置中获取授权码。

#### 阿里云企业邮箱

```bash
-e EMAIL_HOST="smtp.qiye.aliyun.com" \
-e EMAIL_PORT="465" \
-e EMAIL_USERNAME="your-email@your-domain.com" \
-e EMAIL_PASSWORD="your-password" \
-e EMAIL_USE_SSL=true
```

### 完整配置示例

以下是一个包含邮件告警配置的完整Docker运行命令示例：

```bash
docker run -d --name dpanel --restart=always \
 -p 80:80 -p 443:443 -p 8807:8080 \
 -e APP_NAME=dpanel \
 -e CONTAINER_ALERT_ENABLED=true \
 -e CONTAINER_ALERT_MONITOR="mysql,nginx,redis" \
 -e EMAIL_ENABLED=true \
 -e EMAIL_HOST="smtp.gmail.com" \
 -e EMAIL_PORT="465" \
 -e EMAIL_USERNAME="your-email@gmail.com" \
 -e EMAIL_PASSWORD="your-app-password" \
 -e EMAIL_FROM="your-email@gmail.com" \
 -e EMAIL_TO="recipient@example.com" \
 -e EMAIL_USE_SSL=true \
 -v /var/run/docker.sock:/var/run/docker.sock \
 -v dpanel:/dpanel \
 dpanel/dpanel:latest
```

或者使用Docker Compose:

```yaml
version: '3'

services:
  dpanel:
    image: dpanel/dpanel:latest
    container_name: dpanel
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dpanel:/dpanel
    ports:
      - 80:80
      - 443:443
      - 8807:8080
    environment:
      - APP_NAME=dpanel
      - CONTAINER_ALERT_ENABLED=true
      - CONTAINER_ALERT_MONITOR=mysql,nginx,redis
      - EMAIL_ENABLED=true
      - EMAIL_HOST=smtp.gmail.com
      - EMAIL_PORT=465
      - EMAIL_USERNAME=your-email@gmail.com
      - EMAIL_PASSWORD=your-app-password
      - EMAIL_FROM=your-email@gmail.com
      - EMAIL_TO=recipient@example.com
      - EMAIL_USE_SSL=true
    restart: always

volumes:
  dpanel:
```

## 飞书告警配置

DPanel支持将容器告警消息推送到飞书群机器人，配置方式如下：

```bash
# 启用飞书告警
-e FEISHU_ENABLED=true \

# 设置飞书机器人Webhook地址
-e FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" \

# 设置飞书安全设置的签名密钥（如果机器人设置了签名验证）
-e FEISHU_SECRET="xxxxxxxxxxxxxx" \
```

### 飞书机器人配置步骤

1. 进入飞书群 -> 点击群设置 -> 群机器人 -> 添加机器人
2. 选择"自定义机器人"
3. 配置机器人：
   - 名称：随意设置，如"容器告警"
   - 安全设置：可选择"签名校验"（获取签名密钥）
   - 权限：允许发送消息
4. 创建后获取Webhook地址和签名密钥
5. 将获取到的信息填入环境变量

### 完整配置示例

以下是一个包含飞书告警配置的完整Docker运行命令示例：

```bash
docker run -d --name dpanel --restart=always \
 -p 80:80 -p 443:443 -p 8807:8080 \
 -e APP_NAME=dpanel \
 -e CONTAINER_ALERT_ENABLED=true \
 -e CONTAINER_ALERT_MONITOR="mysql,nginx,redis" \
 -e FEISHU_ENABLED=true \
 -e FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" \
 -e FEISHU_SECRET="xxxxxxxxxxxxxx" \
 -v /var/run/docker.sock:/var/run/docker.sock \
 -v dpanel:/dpanel \
 dpanel/dpanel:latest
```

## 企业微信告警配置

DPanel支持将容器告警消息推送到企业微信群机器人，配置方式如下：

```bash
# 启用企业微信告警
-e WECHATWORK_ENABLED=true \

# 设置企业微信机器人Webhook地址
-e WECHATWORK_WEBHOOK_URL="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" \
```

### 企业微信机器人配置步骤

1. 进入企业微信群聊 -> 点击右上角三个点 -> 添加群机器人
2. 选择"新创建一个机器人"
3. 配置机器人：
   - 名称：随意设置，如"容器告警"
   - 描述：如"监控容器状态"
   - 头像：可选
4. 创建后获取Webhook地址（Key）
5. 将获取到的Webhook地址填入环境变量

### 完整配置示例

以下是一个包含企业微信告警配置的完整Docker运行命令示例：

```bash
docker run -d --name dpanel --restart=always \
 -p 80:80 -p 443:443 -p 8807:8080 \
 -e APP_NAME=dpanel \
 -e CONTAINER_ALERT_ENABLED=true \
 -e CONTAINER_ALERT_MONITOR="mysql,nginx,redis" \
 -e WECHATWORK_ENABLED=true \
 -e WECHATWORK_WEBHOOK_URL="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" \
 -v /var/run/docker.sock:/var/run/docker.sock \
 -v dpanel:/dpanel \
 dpanel/dpanel:latest
```

## 常见问题

**Q: 如何查看告警功能是否正常工作？**  
A: 查看容器日志 `docker logs dpanel`，应该能看到容器告警功能初始化的相关信息。

**Q: 为什么我启用了告警但收不到通知？**  
A: 请检查以下几点：
1. 确认 `CONTAINER_ALERT_ENABLED=true` 已设置
2. 如果使用钉钉通知，确认 `DINGTALK_ENABLED=true` 已设置
3. 确认钉钉的Webhook地址和签名密钥格式正确
4. 检查容器日志是否有相关错误信息

**Q: 如何测试告警是否有效？**  
A: 可以启动一个测试容器并设置其健康检查始终失败，或强制退出一个被监控的容器。

**Q: 如何在构建自己的镜像时永久配置这些告警参数？**  
A: 修改Dockerfile中对应的环境变量值，例如：
```dockerfile
# 修改前
ENV CONTAINER_ALERT_ENABLED=false

# 修改后
ENV CONTAINER_ALERT_ENABLED=true
ENV CONTAINER_ALERT_MONITOR="mysql,nginx"
```

**Q: 邮件告警配置正确但收不到邮件怎么办？**  
A: 请检查以下几点：
1. 确认 `EMAIL_ENABLED=true` 已设置
2. 检查SMTP服务器地址和端口是否正确
3. 检查用户名和密码是否正确（有些邮件服务需要使用授权码而非普通密码）
4. 如果使用Gmail等服务，可能需要开启"不安全应用"访问权限或使用应用专用密码
5. 检查容器日志中是否有邮件发送的错误信息

**Q: 使用Office 365或Exchange邮箱无法发送邮件怎么办？**  
A: Office 365和某些Exchange服务器可能需要特殊的认证方式，可以尝试：
1. 使用应用密码而非普通密码
2. 确认邮箱账户已启用SMTP访问权限
3. 检查是否有网络或防火墙限制

**Q: 为什么会收到重复的告警邮件？**  
A: 这可能是因为：
1. 容器反复重启或状态频繁变化
2. 多个监控实例同时监控相同的容器
3. 设置了过短的检查间隔

可以通过调整 `CONTAINER_ALERT_INTERVAL` 或使用更精确的 `CONTAINER_ALERT_MONITOR` 配置来减少重复告警。

**Q: 飞书机器人配置正确但收不到通知怎么办？**  
A: 请检查以下几点：
1. 确认 `FEISHU_ENABLED=true` 已设置
2. 检查Webhook地址是否正确，需要完整的URL包含https://
3. 如果启用了签名校验，确保 `FEISHU_SECRET` 正确填写
4. 确认机器人在群内有足够的权限发送消息
5. 检查容器日志中是否有飞书消息发送的错误信息

**Q: 企业微信机器人收不到告警通知怎么办？**  
A: 可能的原因包括：
1. 确认 `WECHATWORK_ENABLED=true` 已设置
2. Webhook地址可能有误，应以 `https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=` 开头
3. 企业微信机器人可能有频率限制，短时间内不要发送过多消息
4. 检查容器日志，查看是否有相关错误

**Q: 如何同时启用多种告警方式？**  
A: 您可以同时启用多种告警方式，例如：
```bash
docker run -d --name dpanel \
 -e CONTAINER_ALERT_ENABLED=true \
 -e DINGTALK_ENABLED=true \
 -e DINGTALK_WEBHOOK_URL="https://oapi.dingtalk.com/robot/send?access_token=xxxx" \
 -e EMAIL_ENABLED=true \
 -e EMAIL_HOST="smtp.example.com" \
 -e EMAIL_USERNAME="user@example.com" \
 -e EMAIL_PASSWORD="password" \
 -e EMAIL_TO="recipient@example.com" \
 -e FEISHU_ENABLED=true \
 -e FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/xxx" \
 -e WECHATWORK_ENABLED=true \
 -e WECHATWORK_WEBHOOK_URL="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx" \
 ... 其他配置
```

**Q: 启动时报错"Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work"怎么办？**  
A: 这是因为SQLite驱动需要CGO支持。确保在编译时设置`CGO_ENABLED=1`：
```bash
export CGO_ENABLED=1  # Linux/macOS
set CGO_ENABLED=1     # Windows
go build -o runtime/dpanel main.go
```
或者使用本文档推荐的[Docker多阶段构建](#docker多阶段构建推荐)方式，它会自动处理CGO问题。

**Q: 启动容器报错"/app/server/dpanel: not found"怎么办？**
A: 这通常是因为二进制文件没有正确编译或复制到容器中。有两种解决方法：
1. 使用本文档推荐的多阶段构建方式 (Dockerfile.multi)
2. 检查您的Dockerfile中复制二进制文件的路径是否正确

**Q: 编译时遇到Docker API相关错误怎么办？**  
A: 请参阅上面的[解决兼容性问题](#解决兼容性问题)部分。Docker SDK版本不同可能会导致API不兼容，我们提供了解决方案。

**Q: 在Alpine容器中运行报错"no such file or directory"怎么办？**  
A: 确保Alpine容器中安装了SQLite：
```bash
apk add --no-cache sqlite
```

**Q: 交叉编译时出现"gcc: command not found"错误怎么办？**  
A: 需要安装对应平台的交叉编译工具链：
```bash
# Ubuntu/Debian
apt-get install -y gcc-aarch64-linux-gnu  # ARM64
apt-get install -y gcc-arm-linux-gnueabihf  # ARMv7

# 或使用musl交叉编译工具链（macOS）
brew tap messense/macos-cross-toolchains
brew install x86_64-linux-musl aarch64-unknown-linux-musl
```

更多问题请参阅官方文档或社区支持。