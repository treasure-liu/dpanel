# Docker容器错误码参考指南

本文档提供了Docker容器常见错误码的详细说明，用于帮助DPanel用户理解容器告警信息并进行问题排查。

## 错误码概述

当容器异常退出时，Docker会返回一个退出码，该退出码可以帮助诊断问题的根本原因。这些退出码遵循Linux信号约定：

- **0**: 正常退出，没有错误
- **1-127**: 应用程序自定义错误码
- **128+N**: 容器被信号N终止，例如128+9(SIGKILL)=137表示由KILL信号终止

## 常见错误码详解

| 错误码 | 分类 | 说明 | 可能原因 | 排查方法 |
|-------|------|------|---------|---------|
| **0** | 正常退出 | 容器正常结束执行 | 任务完成或短期容器预期行为 | 通常无需处理 |
| **1** | 一般错误 | 通用错误状态 | 应用程序内部错误、配置错误 | 检查容器日志、应用配置 |
| **2** | 命令错误 | 命令语法错误 | 启动命令参数错误 | 检查Dockerfile的CMD/ENTRYPOINT |
| **125** | Docker错误 | Docker守护程序错误 | Docker引擎内部问题 | 检查Docker日志 |
| **126** | 命令不可执行 | 找到命令但无法执行 | 文件权限不足 | 在镜像中执行`chmod +x` |
| **127** | 命令未找到 | 系统找不到指定命令 | 镜像中缺少命令 | 检查镜像构建过程和PATH |
| **137** | SIGKILL终止 | 容器被强制终止 | 内存溢出(OOM)、手动强制停止 | 增加内存限制、检查OOM日志 |
| **139** | 段错误 | 内存访问违规 | 程序试图访问无效内存地址 | 检查应用代码或升级依赖 |
| **143** | SIGTERM终止 | 容器接收到终止信号 | 正常停止容器的标准方式 | 正常行为，检查应用是否正确处理SIGTERM |

## 具体错误码详细说明

### 0 - 正常退出

容器完成其预期工作并正常退出。对于设计为运行短期任务的容器，这是预期行为。

**示例场景**:
- 运行一次性任务的容器
- 执行数据库迁移的容器
- 定时任务容器完成任务后退出

**处理建议**:
- 如果容器应该持续运行，检查应用程序是否有自动退出的逻辑
- 检查主进程是否正常终止

### 1 - 一般错误

通用的非特定错误码，表示应用程序遇到了错误。许多程序使用退出码1表示各种错误情况。

**示例场景**:
- 应用程序遇到配置错误
- 依赖服务不可用
- 文件未找到
- 权限问题

**处理建议**:
- 查看容器日志以获取具体错误消息: `docker logs <container_id>`
- 检查应用程序配置是否正确
- 验证所有依赖服务是否可用

### 2 - 命令错误

命令语法或用法错误。

**示例场景**:
- 容器启动命令参数错误
- 启动脚本中的命令语法错误

**处理建议**:
- 检查容器的启动命令和参数配置
- 查看`Dockerfile`中的`CMD`和`ENTRYPOINT`指令

### 125 - Docker错误

Docker守护进程无法运行容器。

**示例场景**:
- Docker守护程序内部错误
- 容器配置问题(如无效的挂载点或网络配置)

**处理建议**:
- 检查Docker守护程序日志: `journalctl -u docker`
- 验证容器的配置参数，尤其是卷挂载和网络设置

### 126 - 命令不可执行

系统找到了命令但无法执行它。

**示例场景**:
- 执行文件没有执行权限
- 文件系统权限问题

**处理建议**:
- 确保容器中的启动脚本有执行权限
- 在Dockerfile中添加: `RUN chmod +x /path/to/script.sh`

### 127 - 命令未找到

系统找不到指定的命令。

**示例场景**:
- 镜像中没有安装所需的程序
- 命令不在PATH环境变量中
- 引用了错误的可执行文件路径

**处理建议**:
- 确认镜像中是否安装了所需命令
- 检查Dockerfile是否正确安装了依赖
- 验证PATH环境变量配置

### 137 - SIGKILL终止 (128+9)

容器被SIGKILL信号终止，无法捕获或处理这个信号。

**示例场景**:
- 内存溢出(OOM): 容器使用的内存超过限制
- 手动执行`docker kill`命令
- 主机资源不足，系统OOM killer终止进程

**处理建议**:
- 检查是否发生OOM: `dmesg | grep -i 'killed process'`
- 增加容器内存限制: `--memory=2g`
- 检查应用程序内存泄漏
- 对于Java应用，优化JVM内存参数
- 监控容器资源使用情况

### 139 - 段错误 (128+11)

程序访问了非法内存地址，导致SIGSEGV信号触发容器终止。

**示例场景**:
- 程序存在内存访问错误
- 引用空指针或无效内存地址
- 应用程序崩溃
- 依赖库版本不兼容

**处理建议**:
- 检查应用程序日志查找崩溃信息
- 在容器内使用调试工具分析崩溃点
- 更新应用程序或依赖库版本
- 使用核心转储分析崩溃: `docker run --ulimit core=-1 ...`

### 143 - SIGTERM终止 (128+15)

容器收到SIGTERM信号，这是Docker优雅停止容器的标准方式。

**示例场景**:
- 执行`docker stop`命令
- Kubernetes优雅终止Pod
- 系统或服务重启

**处理建议**:
- 确保应用程序正确处理SIGTERM信号以实现优雅关闭
- 这是预期的正常行为，通常不需要特别处理
- 如果应用需要更多时间清理，可以增加停止超时: `docker stop --time=120 <container_id>`

## 健康检查失败

当容器配置了HEALTHCHECK但检查失败时，容器会被标记为`unhealthy`，但不一定会退出。

**示例场景**:
- 应用程序未正常启动
- 应用程序无响应
- 内部服务不可用
- 数据库连接失败
- 磁盘空间不足

**处理建议**:
- 查看健康检查命令: `docker inspect --format='{{.Config.Healthcheck}}' <container_id>`
- 检查健康检查日志: `docker inspect --format='{{json .State.Health}}' <container_id> | jq`
- 登录到容器内部手动执行健康检查命令
- 检查应用服务状态和错误日志
- 验证所有依赖服务状态

## 优化容器告警

### 避免错误告警风暴

1. **重启策略配置**:
   - 只在非正常退出时重启: `--restart=on-failure:3`
   - 带有退避延迟的重启: `--restart-policy=always` 

2. **告警过滤**:
   - 监控特定关键容器: `CONTAINER_ALERT_MONITOR="db,api,web"`
   - 排除测试和开发容器: `CONTAINER_ALERT_EXCLUDE="test-,dev-"` 

3. **减少告警频率**:
   - 增加检查间隔: `CONTAINER_ALERT_INTERVAL=300`
   - 根据需要关闭特定类型告警: `CONTAINER_ALERT_ON_STOP=false`

## 容器问题排查命令

```bash
# 查看容器日志
docker logs --tail=100 <container_id>

# 查看容器详细信息
docker inspect <container_id>

# 查看容器资源使用情况
docker stats <container_id>

# 进入运行中的容器排查问题
docker exec -it <container_id> /bin/sh

# 查看系统OOM日志
dmesg | grep -i "out of memory"

# 查看所有容器退出状态
docker ps -a --format "table {{.Names}}\t{{.Image}}\t{{.Status}}"
```

## 容器设置最佳实践

1. **内存限制设置**:
   ```bash
   docker run --memory=2g --memory-swap=2g ...
   ```

2. **健康检查配置**:
   ```Dockerfile
   HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
     CMD curl -f http://localhost/health || exit 1
   ```

3. **优雅关闭超时**:
   ```bash
   docker run --stop-timeout=120 ...
   ```

4. **资源监控**:
   - 启用Docker监控
   - 使用Prometheus收集容器指标
   - 使用Grafana可视化容器资源使用

## 相关环境变量配置

DPanel提供了以下环境变量来定制容器告警行为:

```
# 启用容器告警功能
CONTAINER_ALERT_ENABLED=true

# 监控指定容器，逗号分隔，留空则监控所有
CONTAINER_ALERT_MONITOR="mysql,nginx,redis"

# 排除指定容器，即使启用监控所有
CONTAINER_ALERT_EXCLUDE="test-,dev-"

# 告警事件配置
CONTAINER_ALERT_ON_STOP=false   # 停止事件
CONTAINER_ALERT_ON_DIE=true     # 异常退出
CONTAINER_ALERT_ON_OOM=true     # 内存溢出
CONTAINER_ALERT_ON_HEALTH=true  # 健康检查失败

# 检查间隔（秒）
CONTAINER_ALERT_INTERVAL=60
```

## 告警日志查看与配置

### 查看告警日志

DPanel的告警消息和发送状态可以通过Docker日志查看：

```bash
# 查看实时日志，包括告警检测和发送状态
docker logs -f dpanel

# 只查看告警相关日志
docker logs dpanel | grep -i "告警"

# 查看最近100条日志
docker logs --tail=100 dpanel

# 查看特定时间范围内的日志
docker logs --since="2023-05-01T00:00:00" --until="2023-05-02T00:00:00" dpanel

# 查看通知发送失败信息
docker logs dpanel | grep "通知发送失败"

# 查看可能的错误原因
docker logs dpanel | grep "可能原因"
```

### 日志文件配置

DPanel支持将日志同时输出到控制台和日志文件，并提供内置的日志轮转功能，可以通过以下环境变量配置：

```bash
# 基本日志配置
-e LOG_LEVEL=info \          # 日志级别：debug, info, warn, error
-e LOG_FORMAT=text \         # 日志格式：text, json
-e LOG_TO_FILE=true \        # 是否写入日志文件
-e LOG_FILE_PATH=/var/log/dpanel/alert.log \  # 日志文件路径

# 日志轮转配置
-e LOG_ROTATE=true \         # 是否启用日志轮转
-e LOG_ROTATE_DAILY=true \   # 是否每日轮转
-e LOG_MAX_SIZE=100 \        # 单个日志文件最大大小(MB)
-e LOG_MAX_BACKUPS=7 \       # 保留的旧日志文件数量
-e LOG_MAX_AGE=30 \          # 日志文件保留天数
```

### 日志文件挂载

为了确保告警日志不会因容器重启而丢失，建议将日志目录挂载到主机：

```bash
# 在docker run命令中添加日志目录挂载
docker run -d --name dpanel \
  -v /path/on/host/logs:/var/log/dpanel \
  -e LOG_TO_FILE=true \
  ... 其他参数 ... \
  dpanel/dpanel:latest
```

在Docker Compose中配置：

```yaml
version: '3'

services:
  dpanel:
    image: dpanel/dpanel:latest
    container_name: dpanel
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dpanel-data:/dpanel
      - ./logs:/var/log/dpanel    # 日志目录挂载
    environment:
      - LOG_TO_FILE=true
      - LOG_FILE_PATH=/var/log/dpanel/alert.log
      - LOG_ROTATE=true
      - LOG_ROTATE_DAILY=true
      - LOG_MAX_SIZE=100
      - LOG_MAX_BACKUPS=7
      - LOG_MAX_AGE=30
      ... 其他配置 ...
```

### 通知错误分析

当通知发送失败时，DPanel会自动分析错误原因并提供可能的解决方案：

```bash
# 查看通知错误详情和分析
docker logs dpanel | grep "通知发送失败" -A 5
```

每种通知方式的常见错误及可能原因：

**钉钉通知错误**:
- 网络连接问题：检查网络连接或代理设置
- Webhook地址无效：更新钉钉机器人配置
- 签名验证失败：检查Secret配置
- 发送频率限制：降低发送频率

**邮件通知错误**:
- 无法连接SMTP服务器：检查服务器地址和端口
- SMTP认证失败：检查用户名和密码
- SSL/TLS连接问题：检查EMAIL_USE_SSL设置
- 邮箱地址无效：检查FROM和TO配置

**飞书通知错误**:
- 网络连接问题：检查网络连接或代理
- Webhook地址或签名错误：检查配置
- 发送频率超限：降低发送频率

**企业微信通知错误**:
- 网络连接问题：检查网络设置
- Webhook地址无效：检查key参数
- API调用频率限制：降低发送频率

---

更多Docker和容器问题排查的详细信息，请参考[Docker官方文档](https://docs.docker.com/) 