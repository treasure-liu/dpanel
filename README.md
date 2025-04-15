<h1 align="center">
<img src="https://cdn.w7.cc/dpanel/dpanel-logo.png" alt="DPanel" width="500" />
</h1>
<h4 align="center"> Docker 可视化面板系统，提供完善的 docker 管理功能。 </h4>

<div align="center">

[![GitHub stars](https://img.shields.io/github/stars/donknap/dpanel.svg)](https://github.com/donknap/dpanel) &nbsp;
[![GitHub latest release](https://img.shields.io/github/v/release/donknap/dpanel)](https://github.com/donknap/dpanel/releases) &nbsp;
[![GitHub latest commit](https://img.shields.io/github/last-commit/donknap/dpanel.svg)](https://github.com/donknap/dpanel/commits/master/) &nbsp;
[![Build Status](https://github.com/donknap/dpanel/actions/workflows/release.yml/badge.svg)](https://github.com/donknap/dpanel/actions) &nbsp;
[![Docker Pulls](https://img.shields.io/docker/pulls/dpanel/dpanel)](https://hub.docker.com/r/dpanel/dpanel/tags) &nbsp;
<a href="https://hellogithub.com/repository/c69089b776704985b989f98626de977a" target="_blank"><img src="https://abroad.hellogithub.com/v1/widgets/recommend.svg?rid=c69089b776704985b989f98626de977a&claim_uid=ekhLfDOxR5U0mVw&theme=small" alt="Featured｜HelloGitHub" /></a>

[**官网**](https://dpanel.cc/) &nbsp; |
&nbsp; [**演示**](https://dpanel.park1991.com) &nbsp; |
&nbsp; [**文档**](https://dpanel.cc/#/zh-cn/install/docker) &nbsp; |
&nbsp; [**Pro版**](https://dpanel.cc/#/zh-cn/manual/pro) &nbsp; |
&nbsp; [**交流群**](https://qm.qq.com/q/2v4x9x8q4k) &nbsp; |
&nbsp; [**赞助**](https://afdian.com/a/dpanel) &nbsp;

</div>

<p>
   <a href="README_EN.md"> English </a> | 中文
</p>

### Pro 版

Pro 版仅是社区版的一个增强和补充，对于通用的、广泛的功能需求不会收录到 Pro 版中。
针对社区版中的部分功能进行强化、升级或是一些极其个性化的需求功能。

如果你热衷于开源产品，并且想获得更多有意思的新特性， Pro 版是适合你的，感谢支持与厚爱。

🚀🚀🚀 [功能介绍及对比](http://dpanel.cc/#/zh-cn/manual/pro?id=%e4%bb%b7%e6%a0%bc%e5%8f%8a%e5%8a%9f%e8%83%bd) 🚀🚀🚀


### 开始使用

> [!IMPORTANT]  
> macos 下需要先将 docker.sock 文件 link 到 /var/run/docker.sock 目录中 \
> ln -s -f /Users/用户/.docker/run/docker.sock  /var/run/docker.sock

> 国内镜像 \
> registry.cn-hangzhou.aliyuncs.com/dpanel/dpanel:latest \
> registry.cn-hangzhou.aliyuncs.com/dpanel/dpanel:lite

#### 标准版

```
docker run -d --name dpanel --restart=always \
 -p 80:80 -p 443:443 -p 8807:8080 -e APP_NAME=dpanel \
 -v /var/run/docker.sock:/var/run/docker.sock -v dpanel:/dpanel \
 dpanel/dpanel:latest 
```

#### lite 版

lite 版去掉了域名转发相关，需要自行转发域名绑定容器，不需要绑定 80 及 443 端口

```
docker run -d --name dpanel --restart=always \
 -p 8807:8080 -e APP_NAME=dpanel \
 -v /var/run/docker.sock:/var/run/docker.sock -v dpanel:/dpanel \
 dpanel/dpanel:lite
```

#### 容器告警通知

DPanel 支持容器状态告警通知功能，可以监控容器的异常退出、OOM、健康检查失败等情况，并通过钉钉机器人发送告警通知。

##### 钉钉告警配置

```
docker run -d --name dpanel --restart=always \
 -p 8807:8080 -e APP_NAME=dpanel \
 -e DINGTALK_ENABLED=true \
 -e DINGTALK_WEBHOOK_URL="https://oapi.dingtalk.com/robot/send?access_token=xxxx" \
 -e DINGTALK_SECRET="SEC000xxxxx" \
 -v /var/run/docker.sock:/var/run/docker.sock -v dpanel:/dpanel \
 dpanel/dpanel:lite
```

##### 容器告警配置

```
docker run -d --name dpanel --restart=always \
 -p 8807:8080 -e APP_NAME=dpanel \
 -e CONTAINER_ALERT_ENABLED=true \
 -e CONTAINER_ALERT_MONITOR="mysql,nginx,app" \
 -e CONTAINER_ALERT_EXCLUDE="temp,test" \
 -e CONTAINER_ALERT_ON_STOP=false \
 -e CONTAINER_ALERT_ON_DIE=true \
 -e CONTAINER_ALERT_ON_OOM=true \
 -e CONTAINER_ALERT_ON_HEALTH=true \
 -e CONTAINER_ALERT_INTERVAL=60 \
 -v /var/run/docker.sock:/var/run/docker.sock -v dpanel:/dpanel \
 dpanel/dpanel:lite
```

#### 告警配置参数说明

| 环境变量 | 说明 | 默认值 |
| --- | --- | --- |
| DINGTALK_ENABLED | 是否启用钉钉告警 | false |
| DINGTALK_WEBHOOK_URL | 钉钉机器人Webhook地址 | "" |
| DINGTALK_SECRET | 钉钉机器人安全设置的签名密钥 | "" |
| CONTAINER_ALERT_ENABLED | 是否启用容器告警 | false |
| CONTAINER_ALERT_MONITOR | 需要监控的容器名称，多个用逗号分隔，为空则监控所有容器 | "" |
| CONTAINER_ALERT_EXCLUDE | 排除监控的容器名称，多个用逗号分隔 | "" |
| CONTAINER_ALERT_ON_STOP | 容器停止时是否告警 | false |
| CONTAINER_ALERT_ON_DIE | 容器异常退出时是否告警 | true |
| CONTAINER_ALERT_ON_OOM | 容器OOM时是否告警 | true |
| CONTAINER_ALERT_ON_HEALTH | 容器健康检查失败时是否告警 | true |
| CONTAINER_ALERT_INTERVAL | 健康检查间隔（秒） | 60 |

#### 集成脚本

> 支持 Debian Ubuntu Alpine，其它发行版未进行测试，请提交 Issue

```
curl -sSL https://dpanel.cc/quick.sh -o quick.sh && sudo bash quick.sh
```

#### 为爱发电

DPanel 是一个开源软件。

如果此项目对你所有帮助，并希望我继续下去，请考虑赞助我为爱发电！感谢所有的爱和支持。

https://afdian.com/a/dpanel

#### 交流群

QQ: 837583876

<img src="https://github.com/donknap/dpanel-docs/blob/master/storage/image/qq.png?raw=true" width="300" />

#### 感谢贡献人员

[![Contributors](https://contrib.rocks/image?repo=donknap/dpanel)](https://github.com/donknap/dpanel/graphs/contributors)

#### 界面预览

###### pro 自定义皮肤

![pro-1](https://cdn.w7.cc/dpanel/pro-1.png)

###### 概览
![home.png](https://raw.githubusercontent.com/donknap/dpanel-docs/master/storage/image/home.png)
###### 容器管理
![app-list.png](https://raw.githubusercontent.com/donknap/dpanel-docs/master/storage/image/app-list.png)
###### 文件管理
![app-file.png](https://raw.githubusercontent.com/donknap/dpanel-docs/master/storage/image/app-file.png)
###### 镜像管理
![image-list.png](https://raw.githubusercontent.com/donknap/dpanel-docs/master/storage/image/image-list.png)
###### 创建镜像
![image-create.png](https://raw.githubusercontent.com/donknap/dpanel-docs/master/storage/image/image-create.png)
###### 创建Compose
![compose-create.png](https://raw.githubusercontent.com/donknap/dpanel-docs/master/storage/image/compose-create.png)
###### 部署Compose
![compose-deploy.png](https://raw.githubusercontent.com/donknap/dpanel-docs/master/storage/image/compose-deploy.png)
###### 系统管理
![system-basic.png](https://raw.githubusercontent.com/donknap/dpanel-docs/master/storage/image/system-basic.png)

#### 相关仓库

- 镜像构建基础模板 https://github.com/donknap/dpanel-base-image
- 文档 https://github.com/donknap/dpanel-docs

#### 相关组件

- Rangine 开发框架 https://github.com/we7coreteam/w7-rangine-go-skeleton
- Docker Sdk https://github.com/docker/docker
- React & UmiJs
- Ant Design & Ant Design Pro & Ant Design Charts

#### Star History
[![Star History Chart](https://api.star-history.com/svg?repos=donknap/dpanel&type=Timeline)](https://star-history.com/#donknap/dpanel&Timeline)
