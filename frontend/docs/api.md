# MC Launcher API 文档

Base URL: `http://localhost:9876/api`。响应统一为 `{"ok":true,"data":...}` 或 `{"ok":false,"error":"..."}`。

## 基础

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/ping` | 健康检查 |
| GET | `/info` | 版本、数据目录、账号数量 |

## 版本

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/versions` | 已安装版本列表 |
| POST | `/versions/install` | body: `{"version":"1.20.1"}` |
| POST | `/versions/remove` | body: `{"version":"1.20.1"}` |
| GET / POST | `/versions/verify` | query 或 body 传 `version`，校验 SHA1 |
| POST | `/launch` | body: `{"version":"1.20.1","username":"Steve"}` |

## 账号

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/accounts` | 返回 `{accounts, current}` |
| PUT | `/accounts` | body: `{"uuid":"..."}` 切换当前账号 |
| DELETE | `/accounts?uuid=...` | 删除账号 |
| POST | `/accounts/offline` | body: `{"name":"Steve"}` |
| POST | `/accounts/microsoft/start` | 返回 `userCode / verificationUri / deviceCode / interval / expiresIn` |
| POST | `/accounts/microsoft/poll` | body: `{"deviceCode":"..."}`；未授权返回 `{"pending":true}`，成功返回 `{"pending":false,"account":{...}}` |

## Java

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/java` | 已检测到的 Java（path / major / version） |
| POST | `/java/install` | body: `{"major":17}` 下载 Adoptium JRE |
| PUT | `/java/set` | body: `{"path":"/usr/bin/java"}` |

## 配置

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/config` | 读取配置 |
| PUT | `/config` | 更新配置（maxMemory/minMemory/threads/width/height/javaPath/lang/source） |

## Modrinth

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/modrinth/search?q=&type=&offset=` | 也兼容 `page`；返回 `{hits, total_hits, offset, limit}` |
| GET | `/modrinth/info/{slug}` | 项目详情 |
| GET | `/modrinth/versions?slug=&game_version=&loader=` | 版本列表 |
| POST | `/modrinth/install` | body: `{"slug":"...","type":"mods","game_version":"1.20.1","loader":"fabric"}` |

## 服务器列表

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/servers` | `[{name, address}]` |
| POST | `/servers` | body: `{"address":"mc.example.com","name":"示例"}` |
| DELETE | `/servers/{address}` | 按地址删除 |

## 示例

```bash
curl http://localhost:9876/api/versions

curl -X POST http://localhost:9876/api/versions/install \
  -H "Content-Type: application/json" \
  -d '{"version":"1.20.1"}'

curl -X POST http://localhost:9876/api/launch \
  -H "Content-Type: application/json" \
  -d '{"version":"1.20.1","username":"Steve"}'
```
