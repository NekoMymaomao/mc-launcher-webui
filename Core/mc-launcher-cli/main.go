//go:build !cshared

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/NekoMymaomao/mc-launcher/core"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	// 初始化
	core.Init()

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "install":
		if len(args) < 1 {
			fmt.Println("用法: mc-launcher install <version>")
			os.Exit(1)
		}
		resp, err := core.InstallVersion(args[0], "", nil)
		handleResp(err, resp, "安装", args[0])

	case "launch":
		if len(args) < 1 {
			fmt.Println("用法: mc-launcher launch <version> [username]")
			os.Exit(1)
		}
		username := "Player"
		if len(args) > 1 {
			username = args[1]
		}
		resp, err := core.LaunchVersion(args[0], username)
		handleResp(err, resp, "启动", args[0])

	case "list", "ls":
		versions := core.ListVersions()
		if len(versions) == 0 {
			fmt.Println("没有已安装的版本")
			return
		}
		fmt.Println("已安装版本:")
		for _, v := range versions {
			fmt.Printf("  %s\n", v)
		}

	case "auth":
		if len(args) < 1 {
			fmt.Println("用法: mc-launcher auth <offline|ms|authlib> [参数...]")
			return
		}
		switch args[0] {
		case "offline":
			name := "Player"
			if len(args) > 1 {
				name = args[1]
			}
			acc := core.LoginOffline(name)
			core.AccountUpsert(*acc)
			fmt.Printf("✓ 离线登录: %s\n", acc.Name)

		case "ms":
			acc, err := core.LoginMicrosoft(true, "")
			if err != nil {
				fmt.Fprintf(os.Stderr, "微软登录失败: %v\n", err)
				os.Exit(1)
			}
			core.AccountUpsert(*acc)
			fmt.Printf("✓ 微软登录: %s\n", acc.Name)

		case "authlib":
			if len(args) < 4 {
				fmt.Println("用法: mc-launcher auth authlib <server> <user> <pass>")
				os.Exit(1)
			}
			acc, err := core.LoginAuthlib(args[1], args[2], args[3])
			if err != nil {
				fmt.Fprintf(os.Stderr, "认证失败: %v\n", err)
				os.Exit(1)
			}
			core.AccountUpsert(*acc)
			fmt.Printf("✓ 认证登录: %s\n", acc.Name)

		default:
			fmt.Printf("未知认证方式: %s\n", args[0])
		}

	case "accounts":
		acc := core.AccountCurrent()
		if acc != nil {
			fmt.Printf("当前账号: %s (%s)\n", acc.Name, acc.Type)
		} else {
			fmt.Println("未登录")
		}

	case "java":
		if len(args) < 1 {
			fmt.Println("用法: mc-launcher java <list|install|set> [参数...]")
			return
		}
		switch args[0] {
		case "list":
			javas := core.ScanJava()
			if len(javas) == 0 {
				fmt.Println("未找到 Java")
				return
			}
			fmt.Println("已安装的 Java:")
			for _, j := range javas {
				fmt.Printf("  %s (Java %d)\n", j.Path, j.Major)
			}
		case "install":
			major := 17
			if len(args) > 1 {
				major = parseInt(args[1])
			}
			path, err := core.InstallJava(major)
			if err != nil {
				fmt.Fprintf(os.Stderr, "安装 Java 失败: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✓ 已安装 Java %d: %s\n", major, path)
		case "set":
			if len(args) < 2 {
				fmt.Println("用法: mc-launcher java set <path>")
				return
			}
			core.SetJavaPath(args[1])
			fmt.Printf("✓ 已设置 Java: %s\n", args[1])
		default:
			fmt.Printf("未知命令: %s\n", args[0])
		}

	case "config":
		if len(args) < 1 {
			fmt.Println("用法: mc-launcher config <show|set> [参数...]")
			return
		}
		switch args[0] {
		case "show":
			c := core.GetSettings()
			fmt.Printf("最大内存: %d MB\n", c.MaxMemory)
			fmt.Printf("最小内存: %d MB\n", c.MinMemory)
			fmt.Printf("线程数: %d\n", c.Threads)
			fmt.Printf("宽度: %d\n", c.Width)
			fmt.Printf("高度: %d\n", c.Height)
			fmt.Printf("Java 路径: %s\n", c.JavaPath)
		case "set":
			if len(args) < 3 {
				fmt.Println("用法: mc-launcher config set <key> <value>")
				return
			}
			key := args[1]
			val := args[2]
			c := core.GetSettings()
			switch key {
			case "maxMemory":
				c.MaxMemory = parseInt(val)
			case "minMemory":
				c.MinMemory = parseInt(val)
			case "threads":
				c.Threads = parseInt(val)
			case "width":
				c.Width = parseInt(val)
			case "height":
				c.Height = parseInt(val)
			case "javaPath":
				c.JavaPath = val
			default:
				fmt.Printf("未知配置项: %s\n", key)
				return
			}
			core.SaveSettings(c)
			fmt.Printf("✓ 已设置 %s = %s\n", key, val)
		default:
			fmt.Printf("未知命令: %s\n", args[0])
		}

	case "lang":
		if len(args) < 1 {
			fmt.Println("用法: mc-launcher lang [zh|en]")
			return
		}
		switch args[0] {
		case "zh", "zh-CN":
			core.SetLang("zh")
			fmt.Println("✓ 语言已切换为中文")
		case "en":
			core.SetLang("en")
			fmt.Println("✓ Language switched to English")
		default:
			fmt.Printf("未知语言: %s\n", args[0])
		}

	case "version", "--version", "-v":
		fmt.Println("MC Launcher v2.0.0-dev")

	case "help", "--help", "-h":
		printUsage()

	default:
		fmt.Printf("未知命令: %s\n", cmd)
		fmt.Println("运行 'mc-launcher help' 查看可用命令")
	}
}

func printUsage() {
	fmt.Println(`MC Launcher - Minecraft 启动器 (CLI)

用法:
  mc-launcher <命令> [参数]

命令:
  install <version>    安装 Minecraft 版本
  launch <version>     启动游戏
  list                 列出已安装版本
  auth <method>        认证登录
  accounts             查看当前账号
  java <cmd>           Java 管理
  config <cmd>         配置管理
  lang [zh|en]         切换语言
  version              显示版本
  help                 显示帮助`)
}

func handleResp(err error, resp *core.Response, action, target string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s失败: %v\n", action, err)
		os.Exit(1)
	}
	if !resp.Ok {
		fmt.Fprintf(os.Stderr, "%s失败: %s\n", action, resp.Error)
		os.Exit(1)
	}
	fmt.Printf("✓ %s成功: %s\n", action, target)
}

func parseInt(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	return n
}
