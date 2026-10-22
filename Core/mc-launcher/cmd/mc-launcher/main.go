package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/NekoMymaomao/mc-launcher/core"
)

func main() {
	lang := flag.String("lang", "", "语言: en 或 zh")
	showVersion := flag.Bool("version", false, "显示版本")
	dir := flag.String("dir", "", "数据目录")
	flag.Parse()

	if *lang != "" {
		core.SetLang(*lang)
	}
	if *dir != "" {
		os.Setenv("MC_LAUNCHER_DIR", *dir)
	}

	if err := core.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "初始化失败: %v\n", err)
		os.Exit(1)
	}

	if *showVersion {
		fmt.Printf("MC Launcher v2.0.0\n")
		fmt.Printf("数据目录: %s\n", core.GetBaseDir())
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		return
	}

	switch args[0] {
	case "install":
		require(args, 2, "install <version>")
		resp, err := core.InstallVersion(args[1], "", func(p core.Progress) {
			fmt.Printf("\r[%d/%d] %s          ", p.Done, p.Total, p.Msg)
			if p.Done >= p.Total {
				fmt.Println()
			}
		})
		handle(resp, err, "安装")

	case "launch":
		require(args, 2, "launch <version> [username]")
		user := "Player"
		if len(args) > 2 {
			user = args[2]
		}
		resp, err := core.LaunchVersion(args[1], user)
		handle(resp, err, "启动")

	case "list", "ls":
		vs := core.ListVersions()
		if len(vs) == 0 {
			fmt.Println("没有已安装的版本")
			return
		}
		for _, v := range vs {
			fmt.Println(v)
		}

	case "remove", "rm":
		require(args, 2, "remove <version>")
		if err := core.RemoveVersion(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "删除失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("已删除:", args[1])

	case "verify":
		require(args, 2, "verify <version>")
		ok, msg := core.VerifyVersion(args[1])
		if ok {
			fmt.Println("验证通过:", args[1])
		} else {
			fmt.Fprintln(os.Stderr, "验证失败:", msg)
			os.Exit(1)
		}

	case "auth":
		require(args, 2, "auth <offline|ms|authlib> ...")
		switch args[1] {
		case "offline":
			name := "Player"
			if len(args) > 2 {
				name = args[2]
			}
			core.UpsertAccount(*core.LoginOffline(name))
			fmt.Println("已登录(离线):", name)
		case "ms":
			acc, err := core.LoginMicrosoft(true, "")
			if err != nil {
				fmt.Fprintf(os.Stderr, "微软登录失败: %v\n", err)
				os.Exit(1)
			}
			core.UpsertAccount(*acc)
			fmt.Println("已登录(微软):", acc.Name)
		case "authlib":
			require(args, 5, "auth authlib <server> <user> <pass>")
			acc, err := core.LoginAuthlib(args[2], args[3], args[4])
			if err != nil {
				fmt.Fprintf(os.Stderr, "认证失败: %v\n", err)
				os.Exit(1)
			}
			core.UpsertAccount(*acc)
			fmt.Println("已登录(authlib):", acc.Name)
		default:
			fmt.Fprintln(os.Stderr, "未知认证方式:", args[1])
			os.Exit(1)
		}

	case "accounts":
		accs := core.ListAccounts()
		if len(accs) == 0 {
			fmt.Println("没有账号")
			return
		}
		cur := core.CurrentAccount()
		for _, a := range accs {
			mark := " "
			if cur != nil && cur.UUID == a.UUID {
				mark = "*"
			}
			fmt.Printf("%s %s (%s) %s\n", mark, a.Name, a.Type, a.UUID)
		}

	case "java":
		require(args, 2, "java <list|install|set> ...")
		switch args[1] {
		case "list":
			for _, j := range core.ScanJava() {
				fmt.Printf("%s  Java %d\n", j.Path, j.Major)
			}
		case "install":
			major := 17
			if len(args) > 2 {
				major = atoi(args[2])
			}
			p, err := core.InstallJava(major)
			if err != nil {
				fmt.Fprintf(os.Stderr, "安装 Java 失败: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("已安装 Java:", p)
		case "set":
			require(args, 3, "java set <path>")
			if err := core.SetJavaPath(args[2]); err != nil {
				fmt.Fprintf(os.Stderr, "保存失败: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("已设置 Java:", args[2])
		default:
			fmt.Fprintln(os.Stderr, "未知子命令:", args[1])
			os.Exit(1)
		}

	case "config":
		require(args, 2, "config <show|set> ...")
		c := core.GetSettings()
		switch args[1] {
		case "show":
			fmt.Printf("maxMemory  = %d\n", c.MaxMemory)
			fmt.Printf("minMemory  = %d\n", c.MinMemory)
			fmt.Printf("threads    = %d\n", c.Threads)
			fmt.Printf("width      = %d\n", c.Width)
			fmt.Printf("height     = %d\n", c.Height)
			fmt.Printf("javaPath   = %s\n", c.JavaPath)
			fmt.Printf("source     = %s\n", c.DownloadSource)
			fmt.Printf("baseDir    = %s\n", core.GetBaseDir())
		case "set":
			require(args, 4, "config set <key> <value>")
			key, val := args[2], args[3]
			upd := &core.Settings{}
			switch key {
			case "maxMemory":
				upd.MaxMemory = atoi(val)
			case "minMemory":
				upd.MinMemory = atoi(val)
			case "threads":
				upd.Threads = atoi(val)
			case "width":
				upd.Width = atoi(val)
			case "height":
				upd.Height = atoi(val)
			case "javaPath":
				upd.JavaPath = val
			case "source":
				upd.DownloadSource = val
			default:
				fmt.Fprintln(os.Stderr, "未知配置项:", key)
				os.Exit(1)
			}
			if err := core.SaveSettings(upd); err != nil {
				fmt.Fprintf(os.Stderr, "保存失败: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("%s = %s\n", key, val)
		default:
			fmt.Fprintln(os.Stderr, "未知子命令:", args[1])
			os.Exit(1)
		}

	case "lang":
		require(args, 2, "lang <en|zh>")
		core.SetLang(args[1])
		fmt.Println("语言:", core.GetLang())

	case "version", "--version", "-v":
		fmt.Println("MC Launcher v2.0.0")

	case "help", "--help", "-h":
		printUsage()

	default:
		fmt.Fprintln(os.Stderr, "未知命令:", args[0])
		printUsage()
		os.Exit(1)
	}
}

func handle(resp *core.Response, err error, action string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s失败: %v\n", action, err)
		os.Exit(1)
	}
	if resp == nil {
		fmt.Fprintf(os.Stderr, "%s失败: 空响应\n", action)
		os.Exit(1)
	}
	if !resp.Ok {
		fmt.Fprintf(os.Stderr, "%s失败: %s\n", action, resp.Error)
		os.Exit(1)
	}
	fmt.Printf("%s成功\n", action)
}

func require(args []string, n int, usage string) {
	if len(args) < n {
		fmt.Fprintln(os.Stderr, "用法: mc-launcher "+usage)
		os.Exit(1)
	}
}

func atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func printUsage() {
	fmt.Print(`MC Launcher - Minecraft 启动器

用法: mc-launcher <命令> [参数]

命令:
  install <version>            安装版本
  launch <version> [username]  启动游戏
  list                         列出已安装版本
  remove <version>             删除版本
  verify <version>             校验版本文件
  auth offline [name]          离线登录
  auth ms                      微软设备码登录
  auth authlib <srv> <u> <p>   Authlib 登录
  accounts                     列出账号
  java list|install|set        管理 Java
  config show|set              查看/修改配置
  lang <en|zh>                 切换语言
  version                      版本信息

全局选项:
  -dir <path>    指定数据目录
  -lang <lang>   指定语言
`)
}
