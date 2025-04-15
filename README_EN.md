<h1 align="center">
<img src="https://cdn.w7.cc/dpanel/dpanel-logo.png" alt="DPanel" width="500" />
</h1>
<h4 align="center"> DPanel one of the most lightweight panel for docker. </h4>

<div align="center">

[![GitHub stars](https://img.shields.io/github/stars/donknap/dpanel.svg)](https://github.com/donknap/dpanel) &nbsp;
[![GitHub latest release](https://img.shields.io/github/v/release/donknap/dpanel)](https://github.com/donknap/dpanel/releases) &nbsp;
[![GitHub latest commit](https://img.shields.io/github/last-commit/donknap/dpanel.svg)](https://github.com/donknap/dpanel/commits/master/) &nbsp;
[![Build Status](https://github.com/donknap/dpanel/actions/workflows/release.yml/badge.svg)](https://github.com/donknap/dpanel/actions) &nbsp;
[![Docker Pulls](https://img.shields.io/docker/pulls/dpanel/dpanel)](https://hub.docker.com/r/dpanel/dpanel/tags) &nbsp;
<a href="https://hellogithub.com/repository/c69089b776704985b989f98626de977a" target="_blank"><img src="https://abroad.hellogithub.com/v1/widgets/recommend.svg?rid=c69089b776704985b989f98626de977a&claim_uid=ekhLfDOxR5U0mVw&theme=small" alt="Featured｜HelloGitHub" /></a>

[**Home**](https://dpanel.cc/) &nbsp; |
&nbsp; [**Demo**](https://dpanel.park1991.com/) &nbsp; |
&nbsp; [**Docs**](https://dpanel.cc/#/zh-cn/install/docker) &nbsp; |
&nbsp; [**Pro Edition**](https://dpanel.cc/#/zh-cn/manual/pro) &nbsp; |
&nbsp; [**Sponsor**](https://afdian.com/a/dpanel) &nbsp;

</div>

### Getting started

> If you need i18n support please contact us to purchase Pro Edition

#### Standard Version

```
docker run -it -d --name dpanel --restart=always \
 -p 80:80 -p 443:443 -p 8807:8080 -e APP_NAME=dpanel \
 -v /var/run/docker.sock:/var/run/docker.sock -v dpanel:/dpanel \
 dpanel/dpanel:latest 
```

#### Lite Version

The lite version removes domain forwarding-related features, no need to bind ports 80 and 443.

```
docker run -it -d --name dpanel --restart=always \
 -p 8807:8080 -e APP_NAME=dpanel \
 -v /var/run/docker.sock:/var/run/docker.sock -v dpanel:/dpanel \
 dpanel/dpanel:lite
```

#### Container Alerts

DPanel supports container status alert notifications to monitor abnormal container exits, OOM, health check failures, etc., and send alerts via DingTalk bot.

##### DingTalk Alert Configuration

```
docker run -it -d --name dpanel --restart=always \
 -p 8807:8080 -e APP_NAME=dpanel \
 -e DINGTALK_ENABLED=true \
 -e DINGTALK_WEBHOOK_URL="https://oapi.dingtalk.com/robot/send?access_token=xxxx" \
 -e DINGTALK_SECRET="SEC000xxxxx" \
 -v /var/run/docker.sock:/var/run/docker.sock -v dpanel:/dpanel \
 dpanel/dpanel:lite
```

##### Container Alert Configuration

```
docker run -it -d --name dpanel --restart=always \
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

#### Alert Configuration Parameters

| Environment Variable | Description | Default |
| --- | --- | --- |
| DINGTALK_ENABLED | Enable DingTalk alerts | false |
| DINGTALK_WEBHOOK_URL | DingTalk robot webhook URL | "" |
| DINGTALK_SECRET | DingTalk robot security signature key | "" |
| CONTAINER_ALERT_ENABLED | Enable container alerts | false |
| CONTAINER_ALERT_MONITOR | Container names to monitor (comma-separated, empty for all) | "" |
| CONTAINER_ALERT_EXCLUDE | Container names to exclude (comma-separated) | "" |
| CONTAINER_ALERT_ON_STOP | Alert when container stops | false |
| CONTAINER_ALERT_ON_DIE | Alert when container exits abnormally | true |
| CONTAINER_ALERT_ON_OOM | Alert when container OOMs | true |
| CONTAINER_ALERT_ON_HEALTH | Alert when container health check fails | true |
| CONTAINER_ALERT_INTERVAL | Health check interval (seconds) | 60 |

#### Install Script 

> Tested on Debian and Alpine.

```
curl -sSL https://dpanel.cc/quick.sh -o quick.sh && sudo bash quick.sh
```

#### Thanks Contributors

[![Contributors](https://contrib.rocks/image?repo=donknap/dpanel)](https://github.com/donknap/dpanel/graphs/contributors)


#### Preview

###### overview
![home.png](https://raw.githubusercontent.com/donknap/dpanel-docs/master/storage/image/home.png)
###### container
![app-list.png](https://raw.githubusercontent.com/donknap/dpanel-docs/master/storage/image/app-list.png)
###### file explorer in container
![app-file.png](https://raw.githubusercontent.com/donknap/dpanel-docs/master/storage/image/app-file.png)
###### image
![image-list.png](https://raw.githubusercontent.com/donknap/dpanel-docs/master/storage/image/image-list.png)
###### build image
![image-create.png](https://raw.githubusercontent.com/donknap/dpanel-docs/master/storage/image/image-create.png)
###### create compose task
![compose-create.png](https://raw.githubusercontent.com/donknap/dpanel-docs/master/storage/image/compose-create.png)
###### deploy compose task
![compose-deploy.png](https://raw.githubusercontent.com/donknap/dpanel-docs/master/storage/image/compose-deploy.png)
###### system
![system-basic.png](https://raw.githubusercontent.com/donknap/dpanel-docs/master/storage/image/system-basic.png)

#### Star History
[![Star History Chart](https://api.star-history.com/svg?repos=donknap/dpanel&type=Timeline)](https://star-history.com/#donknap/dpanel&Timeline)
