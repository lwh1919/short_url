package ioc

import (
	"context"
	"fmt"
	"log"
	"short_url/pkg/generator"
	"short_url/rpc/repository/dao"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"github.com/to404hanga/pkg404/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"gorm.io/sharding"
)

func InitDB(l logger.Logger, cmd redis.Cmdable) *gorm.DB {
	type Config struct {
		User                   string `yaml:"user"`
		Password               string `yaml:"password"`
		Host                   string `yaml:"host"`
		Port                   int    `yaml:"port"`
		Database               string `yaml:"database"`
		TablePrefix            string `yaml:"tablePrefix"`
		EnableDBInit           bool   `yaml:"enableDBInit"`
		SlowThreshold          int64  `yaml:"slowThreshold"`
		SkipDefaultTransaction bool   `yaml:"skipDefaultTransaction"`
	}
	var cfg Config
	err := viper.UnmarshalKey("db", &cfg)
	if err != nil {
		panic(err)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: cfg.SkipDefaultTransaction,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // 单数形式表名
			TablePrefix:   cfg.TablePrefix,
		},
		Logger: glogger.New(gormLoggerFunc(l.Debug), glogger.Config{
			SlowThreshold: time.Duration(cfg.SlowThreshold) * time.Nanosecond, // 单位 ns
			LogLevel:      glogger.Info,
		}),
	})
	if err != nil {
		panic(err)
	}

	// 注册分表中间件
	db.Use(sharding.Register(sharding.Config{
		ShardingKey:    "short_url", // 分表键
		NumberOfShards: 62,          // 分表总数
		// 分表算法，按首字符分表
		ShardingAlgorithm: func(columnValue any) (suffix string, err error) {
			key, ok := columnValue.(string)
			if !ok {
				return "", fmt.Errorf("invalid short_url")
			}
			firstChar := string(key[0])
			suffix = fmt.Sprintf("_%s", firstChar)
			return suffix, nil
		},
		// 分表后缀
		ShardingSuffixs: func() (suffixs []string) {
			ret := make([]string, len(generator.BASE62CHARSET))
			for i, char := range generator.BASE62CHARSET {
				ret[i] = fmt.Sprintf("_%s", string(char))
			}
			return ret
		},
	}, "short_url"))

	// 通过配置文件决定启动时是否初始化数据库
	if cfg.EnableDBInit {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		if ok, _ := cmd.SetNX(ctx, "db_init", true, time.Minute).Result(); ok {
			var rows int64
			err := db.WithContext(ctx).Model(&dao.Mark{}).Count(&rows).Error
			if rows == 0 && (err == nil || err.Error() == "Error 1146 (42S02): Table 'short_url.mark' doesn't exist") {
				go func() {
					log.Println("Starting database initialization...")
					dao.InitTables(db)
					log.Println("Database initialization completed.")
				}()
			}
		}
	}

	return db
}

type gormLoggerFunc func(msg string, fields ...logger.Field)

func (g gormLoggerFunc) Printf(s string, i ...interface{}) {
	g(fmt.Sprintf(s, i...))
}
