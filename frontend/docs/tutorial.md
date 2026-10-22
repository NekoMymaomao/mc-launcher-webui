# MC Launcher 使用教程

## 快速开始

1. 下载对应平台的压缩包：Windows 为 `.zip`，Linux 为 `.tar.gz`
2. 解压后得到 `mc-launcher-server`、命令行版 `mc-launcher` 和 `frontend/`
3. 启动服务：`./mc-launcher-server -addr 0.0.0.0:9876`
4. 浏览器打开 `http://localhost:9876`

## 数据目录

默认 `~/.mc-launcher`（Windows 为 `%USERPROFILE%\.mc-launcher`），可用 `-dir` 指定。

```
~/.mc-launcher/
├── settings.json     启动器配置
├── accounts.json     账号
├── servers.json      服务器列表
├── versions/         游戏版本
├── libraries/        依赖库
├── assets/           游戏资源
├── java/             下载的 Java
├── mods/             模组
├── shaderpacks/      光影
├── resourcepacks/    资源包
└── logs/             日志
```

## 界面功能

- **启动**：选择已安装版本并启动，需先安装 Java
- **版本**：安装 / 校验 / 删除版本。安装会下载客户端、依赖库、natives、资源索引与全部资源文件，并逐个校验 SHA1
- **模组**：从 Modrinth 搜索并下载到 `mods/`
- **账号**：离线登录、微软设备码登录、切换与删除账号
- **设置**：内存、下载线程、窗口尺寸、Java 路径、语言
- **文档**：本页与 API 文档

## 微软登录

1. 点击「Microsoft 登录」
2. 页面显示验证地址和一段代码
3. 在浏览器打开该地址，输入代码并完成授权
4. 页面会自动轮询，成功后刷新账号列表

## Java

- 启动时按版本 JSON 的 `javaVersion.majorVersion` 自动选择 Java
- 自动扫描 `JAVA_HOME`、常见安装目录与数据目录下的 `java/`
- 命令行可用 `mc-launcher java install 17` 下载 Temurin JRE

## 命令行

```bash
mc-launcher list
mc-launcher install 1.20.1
mc-launcher launch 1.20.1
mc-launcher verify 1.20.1
mc-launcher auth offline Steve
mc-launcher auth ms
mc-launcher java list
mc-launcher java install 17
mc-launcher config show
mc-launcher lang zh
```

## 常见问题

**提示找不到 Java？** 先执行 `mc-launcher java list`；没有的话执行 `mc-launcher java install 17`。

**启动失败？** 查看服务端控制台输出的启动命令行与 Java 报错。

**打开 9876 是 404？** 说明没找到前端目录，用 `-static` 指定 `frontend/` 的绝对路径。
