# MC Launcher WebUI

浏览器界面的 Minecraft 启动器，后端是纯 Go（无任何第三方依赖），支持 **Linux 与 Windows**。

## 功能

- 版本安装：从官方 manifest 拉取版本 JSON、客户端 jar、libraries、natives、asset index 与全部 asset objects，并做 SHA1 校验
- 版本管理：列出 / 删除 / 校验已安装版本
- 账号：离线登录、微软设备码登录（OAuth 2.0 device code）、authlib 登录
- Java：自动扫描（含 JAVA_HOME、常见安装目录）、自动下载 Adoptium JRE、按版本所需 Java 大版本自动匹配
- 启动：按版本 JSON 的 `arguments`/`minecraftArguments` 组装参数并替换占位符
- Modrinth：搜索、项目详情、版本查询、一键下载安装

## 项目结构

```
Core/mc-launcher/           Go 库（package core）+ CLI
  ├── api.go                对外接口（Init / Settings / 账号 / 语言）
  ├── version.go            版本 manifest 与 JSON 解析
  ├── install.go            三阶段安装流程
  ├── auth.go               离线 / 微软 / authlib 认证
  ├── java.go               Java 扫描与安装
  ├── launch.go             启动参数组装
  ├── modrinth.go           Modrinth API
  └── cmd/mc-launcher/      CLI 入口
backend/                    Web 服务（导入 core，托管 frontend）
frontend/                   Web UI 静态文件
```

## 编译

需要 Go 1.22+。

```bash
# CLI（Windows / Linux 各架构）
cd Core/mc-launcher
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o mc-launcher-linux-amd64 ./cmd/mc-launcher
CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -o mc-launcher-linux-arm64 ./cmd/mc-launcher
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o mc-launcher-windows-amd64.exe ./cmd/mc-launcher

# Web 服务
cd backend
go build -o mc-launcher-server .
```

也可以直接用 GitHub Actions：推送后自动构建，产物在 Actions 的 `mc-launcher-dist` 中（含前端的三平台压缩包）。

## 运行

### Web 界面

```bash
# 数据目录默认 ~/.mc-launcher，可用 -dir 指定
./mc-launcher-server -addr 0.0.0.0:9876

# 浏览器打开
# http://localhost:9876
```

服务会按以下顺序查找前端目录：`-static` 参数 → 可执行文件旁的 `static/` → 当前目录的 `frontend/` → 数据目录下的 `static/`。

### 命令行

```bash
./mc-launcher-linux-amd64 list
./mc-launcher-linux-amd64 install 1.20.1
./mc-launcher-linux-amd64 auth offline Steve
./mc-launcher-linux-amd64 java list
./mc-launcher-linux-amd64 java install 17
./mc-launcher-linux-amd64 launch 1.20.1
./mc-launcher-linux-amd64 config show
./mc-launcher-linux-amd64 verify 1.20.1
```

全局选项：`-dir <路径>`、`-lang <en|zh>`、`-version`。

## 一键安装

```bash
# Linux
bash install-linux.sh

# Windows（cmd 中运行）
install-windows.bat
```

## HTTP API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/ping` | 健康检查 |
| GET | `/api/versions` | 已安装版本 |
| POST | `/api/versions/install` | `{"version":"1.20.1"}` |
| POST | `/api/versions/remove` | `{"version":"1.20.1"}` |
| GET | `/api/versions/verify?version=` | 校验版本文件 |
| POST | `/api/launch` | `{"version":"1.20.1","username":"Steve"}` |
| GET | `/api/accounts` | 账号列表与当前账号 |
| PUT | `/api/accounts` | `{"uuid":"..."}` 切换账号 |
| DELETE | `/api/accounts?uuid=` | 删除账号 |
| POST | `/api/accounts/offline` | `{"name":"Steve"}` |
| POST | `/api/accounts/microsoft` | 触发设备码登录（控制台会打印验证码） |
| GET | `/api/java` | 可用 Java 列表 |
| POST | `/api/java/install` | `{"major":17}` |
| PUT | `/api/java/set` | `{"path":"/usr/bin/java"}` |
| GET/PUT | `/api/config` | 读取 / 修改配置 |
| GET | `/api/modrinth/search?q=&type=&offset=` | 搜索项目 |

## 数据目录布局

```
~/.mc-launcher/
├── settings.json
├── accounts.json
├── versions/<版本>/<版本>.json|.jar
├── libraries/
├── assets/indexes|objects/
├── java/
└── logs/
```

## 说明

- 微软登录使用 OAuth 2.0 设备码流程，终端/服务控制台会输出验证地址与代码。
- 账号以 UUID 区分，可同时保存多个微软与离线账号。
- 仅支持 Linux 与 Windows；其他平台未做适配。
