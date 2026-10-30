// Command server 为 server-go 的启动入口。
package main

import (
	"flag"
	"fmt"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
	"github.com/YGeni/ruoyi-go/server-go/internal/router"
	"github.com/YGeni/ruoyi-go/server-go/internal/scheduler"
	"github.com/YGeni/ruoyi-go/server-go/pkg/db"
	"github.com/YGeni/ruoyi-go/server-go/pkg/logger"
	"github.com/YGeni/ruoyi-go/server-go/pkg/redisx"

	"github.com/gin-gonic/gin"
)

// 启动标识：紫色（256 色 141）ASCII 字样的 Genius。
const (
	colorPurple = "\033[38;5;141m"
	colorReset  = "\033[0m"

	geniusBanner = `
  ____  _____  _   _  ___  _   _  ____
 / ___|| ____|| \ | ||_ _|| | | / ___|
| |  _ |  _|  |  \| | | | | | | \___ \
| |_| || |___ | |\  | | | | |_| |___) |
 \____||_____||_| \_||___| \___/ |____/
`
)

// printGeniusBanner 在依赖就绪、开始监听端口前打印项目标识与常用地址。
func printGeniusBanner(port int) {
	fmt.Printf("%s%s%s", colorPurple, geniusBanner, colorReset)
	fmt.Printf("%s天才管理系统%s  server-go v%s\n", colorPurple, colorReset, config.C.RuoYi.Version)
	// /v3/api-docs 与 /swagger-ui/* 都校验登录态（见 openapi.go 的 authForIframe），
	// 直接裸开会 401，所以这里指向后台「系统工具 → 系统接口」菜单地址：
	// 前端 iframe 会自动把 token 拼上。
	fmt.Printf("  系统接口    http://localhost/tool/swagger（登录后查看，后端 %d）\n", port)
	fmt.Printf("  OpenAPI    http://localhost:%d/v3/api-docs?token=<登录 Token>\n", port)
	fmt.Printf("  健康检查   http://localhost:%d/captchaImage\n", port)
	fmt.Println()
}

func main() {
	cfgPath := flag.String("c", "config/config.yaml", "配置文件路径")
	flag.Parse()

	if err := config.Load(*cfgPath); err != nil {
		panic(err)
	}
	if err := logger.Init(config.C.Log.Level); err != nil {
		panic(err)
	}
	defer logger.Sync()

	if err := db.Init(config.C.Datasource.Master); err != nil {
		logger.Sugar.Fatalf("MySQL 初始化失败: %v", err)
	}
	if err := redisx.Init(config.C.Redis); err != nil {
		logger.Sugar.Fatalf("Redis 初始化失败: %v", err)
	}
	defer redisx.Close()

	// 启动定时任务调度器：等价 Java 侧
	// SysJobServiceImpl 里的 @PostConstruct init()（L44-49）——
	// 应用启动时把 sys_job 里 status='0' 的任务灌进 quartz。
	// 这里同样在监听端口之前启动，失败只记日志不阻断启动
	// （任务表不可用时页面仍应能查列表）。
	scheduler.Start(repository.JobScheduleRepo{})
	defer scheduler.Stop()

	gin.SetMode(config.C.Server.Mode)
	r := router.New()
	addr := fmt.Sprintf(":%d", config.C.Server.Port)
	logger.Sugar.Infof("server-go 启动于 %s（前端 vite 代理目标 http://localhost:%d）", addr, config.C.Server.Port)
	printGeniusBanner(config.C.Server.Port)
	if err := r.Run(addr); err != nil {
		logger.Sugar.Fatalf("服务退出: %v", err)
	}
}
