//go:build !cshared

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func main() {
	lng := flag.String("lang", "", "语言: en 或 zh")
	versionFlag := flag.Bool("version", false, "显示版本")
	flag.Parse()

	if *lng == "zh" || *lng == "zh-CN" {
		curLang = "zh"
	} else if *lng == "en" {
		curLang = "en"
	}

	if *versionFlag {
		fmt.Printf("MC Launcher v2.0.0-dev\n")
		fmt.Printf("Build: Go %s/%s\n", runtimeGOOS, runtimeGOARCH)
		fmt.Printf("Language: %s\n", curLang)
		return
	}

	if len(flag.Args()) == 0 {
		printUsage()
		return
	}

	baseDir = launcherDir()
	ensureDir(baseDir)
	cfg = loadSettings()
	accounts = loadAccounts()

	for _, d := range []string{
		"versions", "libraries",
		"assets/indexes", "assets/objects",
		"java", "logs",
	} {
		ensureDir(filepath.Join(baseDir, d))
	}

	cmd := flag.Arg(0)
	args := flag.Args()[1:]

	switch cmd {
	case "install":
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "用法: mc-launcher install <version>")
			os.Exit(1)
		}
		resp, err := installVersion(args[0], "", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
		if resp.Ok {
			fmt.Printf("已安装: %s\n", args[0])
		} else {
			fmt.Fprintf(os.Stderr, "错误: %s\n", resp.Error)
			os.Exit(1)
		}

	case "launch":
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "用法: mc-launcher launch <version> [username]")
			os.Exit(1)
		}
		username := "Player"
		if len(args) > 1 {
			username = args[1]
		}
		resp, err := launchVersion(args[0], username)
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
		if resp.Ok {
			fmt.Printf("已启动: %s\n", args[0])
		} else {
			fmt.Fprintf(os.Stderr, "错误: %s\n", resp.Error)
			os.Exit(1)
		}

	case "list", "ls":
		versions := listVersions()
		if len(versions) == 0 {
			fmt.Println("没有已安装的版本")
			return
		}
		fmt.Println("已安装版本:")
		for _, v := range versions {
			fmt.Printf("  %s\n", v)
		}

	case "remove", "rm":
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "用法: mc-launcher remove <version>")
			os.Exit(1)
		}
		removeVersion(args[0])
		fmt.Printf("已删除: %s\n", args[0])

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
			acc := loginOffline(name)
			accounts.upsert(*acc)
			fmt.Printf("已登录: %s (%s)\n", acc.Name, acc.UUID)
		case "ms":
			acc, err := loginMicrosoft(true, "")
			if err != nil {
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
				os.Exit(1)
			}
			accounts.upsert(*acc)
			fmt.Printf("已登录: %s (%s)\n", acc.Name, acc.UUID)
		case "authlib":
			if len(args) < 4 {
				fmt.Fprintln(os.Stderr, "用法: mc-launcher auth authlib <server> <user> <pass>")
				os.Exit(1)
			}
			acc, err := loginAuthlib(args[1], args[2], args[3])
			if err != nil {
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
				os.Exit(1)
			}
			accounts.upsert(*acc)
			fmt.Printf("已登录: %s (%s)\n", acc.Name, acc.UUID)
		default:
			fmt.Printf("未知认证方式: %s\n", args[0])
		}

	case "accounts":
		acc := accounts.current()
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
			javas := scanJava()
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
			path, err := installJava(major)
			if err != nil {
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("已安装 Java %d: %s\n", major, path)
		case "set":
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "用法: mc-launcher java set <path>")
				return
			}
			cfg.JavaPath = args[1]
			cfg.save()
			fmt.Printf("已设置 Java: %s\n", args[1])
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
			fmt.Printf("最大内存: %d MB\n", cfg.MaxMemory)
			fmt.Printf("最小内存: %d MB\n", cfg.MinMemory)
			fmt.Printf("线程数: %d\n", cfg.Threads)
			fmt.Printf("宽度: %d\n", cfg.Width)
			fmt.Printf("高度: %d\n", cfg.Height)
			fmt.Printf("Java 路径: %s\n", cfg.JavaPath)
		case "set":
			if len(args) < 3 {
				fmt.Fprintln(os.Stderr, "用法: mc-launcher config set <key> <value>")
				return
			}
			key := args[1]
			val := args[2]
			switch key {
			case "maxMemory":
				cfg.MaxMemory = parseInt(val)
			case "minMemory":
				cfg.MinMemory = parseInt(val)
			case "threads":
				cfg.Threads = parseInt(val)
			case "width":
				cfg.Width = parseInt(val)
			case "height":
				cfg.Height = parseInt(val)
			case "javaPath":
				cfg.JavaPath = val
			default:
				fmt.Printf("未知配置项: %s\n", key)
				return
			}
			cfg.save()
			fmt.Printf("已设置 %s = %s\n", key, val)
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
			curLang = "zh"
			fmt.Println("✓ 语言已切换为中文")
		case "en":
			curLang = "en"
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
	fmt.Println(`MC Launcher - Minecraft 启动器

用法:
  mc-launcher <命令> [参数]

命令:
  install <version>    安装 Minecraft 版本
  launch <version>     启动游戏
  list                 列出已安装版本
  remove <version>     删除版本
  auth <method>        认证登录
  accounts             查看当前账号
  java <cmd>           Java 管理
  config <cmd>         配置管理
  lang [zh|en]         切换语言
  version              显示版本
  help                 显示帮助`)
}

func parseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
