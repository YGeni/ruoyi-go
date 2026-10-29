// Package db 初始化 GORM（MySQL）。
package db

import (
	"time"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/pkg/sqlstats"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// DB 全局数据库句柄。
var DB *gorm.DB

// Init 初始化数据库连接与连接池。
// 连接池参数映射自 application-druid.yml：
//   - minIdle=10        -> SetMaxIdleConns
//   - maxActive=20      -> SetMaxOpenConns
//   - minEvictableIdleTimeMillis=300000 -> SetConnMaxIdleTime(5min)
//   - maxEvictableIdleTimeMillis=900000 -> SetConnMaxLifetime(15min)
func Init(cfg config.DBConfig) error {
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       cfg.DSN(),
		DefaultStringSize:         256,
		DisableDatetimePrecision:  true,
		DontSupportRenameIndex:    true,
		SkipInitializeWithVersion: false,
	}), &gorm.Config{
		// 表名与 Java 实体一致：SysUser -> sys_user（不复数化）
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
		Logger:         logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return err
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxIdleConns(cfg.MinIdle)
	sqlDB.SetMaxOpenConns(cfg.MaxActive)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	sqlDB.SetConnMaxLifetime(15 * time.Minute)
	DB = gormDB
	// 装 SQL 执行统计（druid "SQL 监控" 的 Go 侧替代，见 pkg/sqlstats）。
	// 必须在任何业务查询之前，所以放在 Init 里而不是 main。
	sqlstats.Install(gormDB)
	return nil
}
