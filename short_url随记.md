## 1  [wire用法](https://cloud.tencent.com/developer/article/2331645)

```go
//APP变量注入器
func Init() *App {
	//提供者
    wire.Build(
       //自动依赖
       ......
       //自动匹配填充结构体变量APP
       wire.Struct(new(App), "*"),
    )
    //只做占位
    return new(App)
}
```

返回

```go
func Init() *App {
   ......
    app := &App{
       GrpcServer: server,
       Cron:       cron,
    }
    return app
}
```

## 2 [pflag]([命令行参数解析工具 —— Go Pflag 入门本文通过丰富的代码示例演示了 pflag 的基本功能，读者可以跟随这些示 - 掘金](https://juejin.cn/post/7265329025899085879#heading-1))

```go
func initViperWatch() {
	//获取环境变量cfile     键        默认值             说明
	cfile := pflag.String("config","config/config.yaml", "配置文件路径")
    //pflag.Parse() = 读取命令行参数 + 解析参数值 + 适配类型 + 填充变量
	pflag.Parse()
	// 直接指定文件路径
	viper.SetConfigFile(*cfile)
	viper.WatchConfig()//启用监控但是未重载函数
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
}
```

## 3 ioc

（1）三者关系（new一个类（变量实例化），依赖注入，控制反转）：

​	1）依赖注入是实现控制反转的一种形式；（控制反转(IoC)是设计原则，依赖注入(DI)是具体实现模式）

​	3）new一个类和控制反转显著作用区别：耦合性，解决复杂依赖图、生命周期管理、配置集成等痛点

​			例子：

```go
new一个类
// service/user_service.go
type UserService struct {
    repo *UserRepository
}

func NewUserService() *UserService {
    // 直接创建依赖项
    db := NewMySQLDatabase("localhost:3306")
    return &UserService{
        repo: NewUserRepository(db),
    }
}
==========
// 测试时需要mock数据库
func TestUserService(t *testing.T) {
    // 无法隔离测试 - 会连接真实数据库!
    svc := NewUserService()
    // ...
}
==========
依赖注入：
// service/user_service.go
type UserService struct {
    repo UserRepository // 依赖接口而非具体实现
}

// 通过构造函数注入依赖
func NewUserService(repo UserRepository) *UserService {
    return &UserService{repo: repo}
}

// main.go
func main() {
    // 在应用入口组装依赖
    db := NewMySQLDatabase(cfg.DBUrl)
    repo := NewUserRepository(db)
    svc := NewUserService(repo) // 注入依赖
}
==========
func TestUserService(t *testing.T) {
    mockRepo := new(MockUserRepository) // 实现UserRepository接口
    svc := NewUserService(mockRepo)
    // 可以隔离测试业务逻辑
}
==========
控制反转：
// wire.go (Google Wire)
var Set = wire.NewSet(
    ProvideConfig,
    ProvideMySQLDB,
    ProvideUserRepository,
    ProvideUserService,
)

func InitializeApp() (*App, error) {
    wire.Build(
        Set,
        wire.Struct(new(App), "*"),
    )
    return &App{}, nil
}

// main.go
func main() {
    // 容器自动装配所有依赖
    app, err := InitializeApp()
    if err != nil {
        panic(err)
    }
    app.Run()
}
==========
集中管理的依赖注入
==========
```



## 4 中间件——Gorm Sharding

简单解释：

sharding.Register(sharding.Config{分表键string（根据改字段来计算分表），分表总数int，分表算法函数fuck，分表后缀凭借函数func（创建表时和某些需要全表扫描的操作时被使用），表名称string（匹配表）}）

注册后，使用

db.AutoMigrate(&Mark{})//穿透中间件
db.AutoMigrate(&ShortUrl{})//执行（表名称匹配）

查询时候分表（通常）查询

```go
// 注册分表中间件
	db.Use(sharding.Register(sharding.Config{
		ShardingKey:    "short_url", // 分表键
		NumberOfShards: 62,          // 分表总数
		// 分表算法，按首字符分表
        //注：any等于接口类型
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
```

##  5表的初始化——mark表

问题1：采用mark表的好处

1）提供原子化的全局初始化状态标记，确保分布式环境下数据库表结构创建的完整性、一致性和可追溯性，同时避免分表检查的性能开销

2）版本性：可拓展词条 SchemaVersion string `gorm:"type:varchar(32);not null"` *// 核心版本字段*

```go
	// 通过配置文件决定启动时是否初始化数据库
	if cfg.EnableDBInit {
        //现时ctx
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
        //采用分布式锁
		if ok, _ := cmd.SetNX(ctx, "db_init", true, time.Minute).Result(); ok {
			var rows int64
            //应用同一个ctx时间是继承的
			err := db.WithContext(ctx).Model(&dao.Mark{}).Count(&rows).Error
            //无记录		  //表存在但空	 //表不存在
			if rows == 0 && (err == nil || err.Error() == "Error 1146 (42S02): Table 'short_url.mark' doesn't exist") {
				go func() {
					log.Println("Starting database initialization...")
					dao.InitTables(db)
					log.Println("Database initialization completed.")
				}()
			}
		}
	}

func InitTables(db *gorm.DB) {
	db.AutoMigrate(&Mark{})
	db.AutoMigrate(&ShortUrl{})
	db.WithContext(context.Background()).Create(&Mark{
		Inited: true,
	})
}

type Mark struct {
	Inited bool `gorm:"type:tinyint(1)"`
}

```

## 6数据库的连接：

咳咳，其实是因为我之前的数据库连接都是很简单的，这个项目里面多了好几个我不认识的字段：

```go
func InitDB(l logger.Logger, cmd redis.Cmdable) *gorm.DB {
	type Config struct {
		User                   string `yaml:"user"`
		Password               string `yaml:"password"`
		Host                   string `yaml:"host"`
		Port                   int    `yaml:"port"`
		Database               string `yaml:"database"`
		TablePrefix            string `yaml:"tablePrefix"` //为所有表名添加统一前缀
		EnableDBInit           bool   `yaml:"enableDBInit"`
		SlowThreshold          int64  `yaml:"slowThreshold"`//定义SQL执行时间的警告阈值
		SkipDefaultTransaction bool   `yaml:"skipDefaultTransaction"`
		//MaxOpenConns           int           `yaml:"maxOpenConns"`     // 新增
		//MaxIdleConns           int           `yaml:"maxIdleConns"`     // 新增
		//ConnMaxLifetime        time.Duration `yaml:"connMaxLifetime"`    // 新增
	}
	var cfg Config
	err := viper.UnmarshalKey("db", &cfg)
	if err != nil {
		panic(err)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: cfg.SkipDefaultTransaction,//禁用GORM的自动事务封装
		NamingStrategy: schema.NamingStrategy{//命名策略 
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

	//// 设置连接池参数（带默认值）
	//if cfg.MaxOpenConns > 0 {
	//	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	//}
	//if cfg.MaxIdleConns > 0 {
	//	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	//}
	//if cfg.ConnMaxLifetime > 0 {
	//	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	//}
	//
	//// 连接测试
	//if err := sqlDB.Ping(); err != nil {
	//	panic(fmt.Errorf("数据库连接测试失败: %w", err))
	//}
    db.Use()....
    .....
}
```

注上述代码还用到一个接口桥接：（由于logger包是04h哥的封装包，目前不熟悉，日后补，先了解原理）函数签名匹配是干什么鬼.....

```go
	Logger: glogger.New(gormLoggerFunc(l.Debug), glogger.Config{
			SlowThreshold: time.Duration(cfg.SlowThreshold) * time.Nanosecond, // 单位 ns
			LogLevel:      glogger.Info,
		}),
==============
//函数签名规范
type gormLoggerFunc func(msg string, fields ...logger.Field)
func (g gormLoggerFunc) Printf(s string, i ...interface{}) {
	g(fmt.Sprintf(s, i...))
}

==============
解释：
GORM日志接口要求：
type Interface interface {
    LogMode(LogLevel) Interface
    Info(context.Context, string, ...interface{})
    Warn(context.Context, string, ...interface{})
    Error(context.Context, string, ...interface{})
    Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error)
}
glogger.New 创建了适配器：
func New(writer Writer, config Config) Interface {
    return &logger{
        Writer: writer,
        Config: config,
    }
}

logger实现了GORM的Interface
func (l *logger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
    // 调用您的gormLoggerFunc
    l.Writer.Printf(formatLog(...))
}
```

## 7 [etcd](https://www.liwenzhou.com/posts/Go/etcd/)

目前还不知道etcd在代码中的体现和作用，日后研究04h哥的第三方库grpcx

但是在配置上：

```go
// 本地配置和etcd配置
func InitService(ecli *clientv3.Client, repo repository.ShortUrlRepository, l logger.Logger) service.ShortUrlService {
    type Config struct {
       Suffix string `yaml:"suffix"`
    }
    cfg := &Config{
       Suffix: "_TO404HANGA",
    }
    if err := viper.UnmarshalKey("short_url", &cfg); err != nil {
       panic(err)
    }

    // // 获取 etcd 键值对 "weights"，并将其转换为 weights 切片
    // ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    // defer cancel()
    // resp, err := ecli.Get(ctx, "weights")
    // if err != nil {
    //     panic(err)
    // }
    // kv := resp.Kvs[0]
    // weightStr := string(kv.Value)
    // weights := make([]int, 0, 6)
    // for _, w := range strings.Split(weightStr, ",") {
    //     if i, err := strconv.Atoi(w); err == nil {
    //        weights = append(weights, i)
    //     }
    // }
    weights := viper.GetIntSlice("short_url.weights")
    svc := service.NewCachedShortUrlService(repo, l, cfg.Suffix, weights)

    // // 监听 etcd 键值对的变化并更新 weights
    // go func() {
    //     watchChan := ecli.Watch(context.Background(), "weights")
    //     for resp := range watchChan {
    //        for _, ev := range resp.Events {
    //           if string(ev.Kv.Key) == "weights" {
    //              weightStr = string(ev.Kv.Value)
    //              weights := make([]int, 0, 6)
    //              for _, w := range strings.Split(weightStr, ",") {
    //                 if i, err := strconv.Atoi(w); err == nil {
    //                    weights = append(weights, i)
    //                 }
    //              }
    //              svc.Weights = weights
    //           }
    //        }
    //     }
    // }()

    return svc
}
```

##  8 grpc 

init部分i

<img src="C:\Users\linweihao\AppData\Roaming\Typora\typora-user-images\image-20250814155552178.png" alt="image-20250814155552178" style="zoom:60%;" />

```go
func InitGrpcxServer(shortUrl *grpc2.ShortUrlServiceServer, ecli *clientv3.Client, l logger.Logger) *grpcx.Server {
    type Config struct {
       Port     int    `yaml:"port"`
       EtcdAddr string `yaml:"etcdAddr"`
       EtcdTTL  int64  `yaml:"etcdTTL"` //// 服务注册有效期(秒)
    }
    var cfg Config
    if err := viper.UnmarshalKey("grpc.server", &cfg); err != nil {
       panic(err)
    }
    //空的 gRPC 服务器容器
    server := grpc.NewServer()
    //使用了protobuf生成的注册函数
    shortUrl.Register(server)
    return &grpcx.Server{
       Server:     server,
       Port:       cfg.Port,
       EtcdClient: ecli,
       Name:       "short_url",
       EtcdTTL:    cfg.EtcdTTL,
       L:          l,
    }
}
func (s *ShortUrlServiceServer) Register(server grpc.ServiceRegistrar) {
	short_url_v1.RegisterShortUrlServiceServer(server, s)
}

```









## 9 cronjob包的定时清理调度图

<img src="C:\Users\linweihao\AppData\Roaming\Typora\typora-user-images\image-20250814164539146.png" alt="image-20250814164539146" style="zoom:50%;" />

<img src="C:\Users\linweihao\AppData\Roaming\Typora\typora-user-images\image-20250814165633881.png" alt="image-20250814165633881" style="zoom:50%;" />

需要添加c.start()才会开始



## 10 日志部分





## 11redis及cache（缓存）

redis初始化部分：

**面向接口设计**是一种编程范式：

- 定义抽象契约（接口）
- 实现与使用分离
- 依赖抽象而非具体实现

也就是说：

1. **定义契约**：先定义 `ShortUrlCache`接口（能做什么）

2. **实现分离**：编写多个实现（怎么做）

3. **依赖抽象**：业务代码只依赖接口

4. **运行时绑定**：由IoC容器决定具体实现

   

   `redis.Cmdable`是一个**接口类型**，而 `redis.Client`是该接口的**具体实现**。代码中返回 `redis.Cmdable`而非 `redis.Client`，主要是基于**接口抽象、灵活性和可扩展性**的设计考量

   面向接口实现之一

```go
func InitRedis() redis.Cmdable {
	
	type Config struct {
	}
	cfg := Config{}
	//日后只需要修改这和拓展即可
	cmd := redis.NewClient(&redis.Options{
		
	})
	return cmd
}

```

面向接口具体实现：

```go
实现编写
type RedisShortUrlCache struct {
	cmd        redis.Cmdable
	prefix     string
	expiration time.Duration
}
实现在编译阶段能确保接口匹配
var _ ShortUrlCache = (*RedisShortUrlCache)(nil)

func NewRedisShortUrlCache(cmd redis.Cmdable, prefix string, expiration time.Duration) ShortUrlCache {
	return &RedisShortUrlCache{
		cmd:        cmd,
		prefix:     prefix,
		expiration: expiration,
	}
}

func (r *RedisShortUrlCache) Get(ctx context.Context, shortUrl string) (originUrl string, err error) {
	return r.cmd.Get(ctx, r.key(shortUrl)).Result()
}

func (r *RedisShortUrlCache) Set(ctx context.Context, shortUrl string, originUrl string) error {
	_, err := r.cmd.Set(ctx, r.key(shortUrl), originUrl, r.expiration+time.Duration(rand.IntN(7201)-3600)).Result() // 随机加减一小时过期时间
	return err
}

func (r *RedisShortUrlCache) Del(ctx context.Context, shortUrl string) error {
	return r.cmd.Del(ctx, r.key(shortUrl)).Err()
}

func (r *RedisShortUrlCache) Refresh(ctx context.Context, shortUrl string) error {
	return r.cmd.Expire(ctx, r.key(shortUrl), r.expiration+time.Duration(rand.IntN(7201)-3600)).Err() // 随机加减一小时过期时间
}

func (r *RedisShortUrlCache) key(shortUrl string) string {
	return r.prefix + ":" + shortUrl
}
定义抽象契约（接口）
type ShortUrlCache interface {
	Get(ctx context.Context, shortUrl string) (originUrl string, err error)
	Set(ctx context.Context, shortUrl string, originUrl string) error
	Del(ctx context.Context, shortUrl string) error
	Refresh(ctx context.Context, shortUrl string) error
}

```





12 server层随记

1）问题1，会不会永久阻塞：不会，生成锻炼算法低，且ctx上下文控制

2）问题2：为什么要设置无条件for循环：

```go

func (s *CachedShortUrlService) Create(ctx context.Context, originUrl string) (string, error) {
	baseSuffix := ""
	for {
		shortUrl := generator.GenerateShortUrl(originUrl, baseSuffix, s.Weights)
		err := s.repo.InsertShortUrl(ctx, shortUrl, originUrl)
		switch err {
		case nil, repository.ErrUniqueIndexConflict://短链值重复（哈希碰撞）
			return shortUrl, nil
		case repository.ErrPrimaryKeyConflict://原始URL重复（幂等操作）
			baseSuffix += s.suffix
		default:
			return "", err
		}
	}
}
```

13数据库层：这一层是整个项目的精华所在，建议直接去阅读源代码

由上到底：



缓存热点的实现：

补充前言：先不说lru淘汰机制（中文译为"最近最少使用"）（lru.cache是本地缓存包），多级缓存；目前我学习的是比较简单的工具包（调库侠），更难度是什么是热点（高频，业务价值高）？怎么识别？怎么提前预判？怎么淘汰？怎么多节点一致？

```go
ps：web路由层采用了合并相同请求，那数据库需要合并请求吗？;ans:ai说是单体项目不需要，分布式需要，但是我有感觉分布式能有多少个服务器，也没多大影响？其次对于三级缓存验证有三个版本待测试：1双重合并 2单重合并 2双重合并，但是只保护数据库层，不保护缓存 
result, err, _ := h.requestGroup.Do(shortUrl, func() (interface{}, error) {
    resp, err := h.svc.GetOriginUrl(ctx, &short_url_v1.GetOriginUrlRequest{
        ShortUrl: shortUrl,
    })
    return resp.GetOriginUrl(), err
})
repo
func (c *CachedShortUrlRepository) GetOriginUrlByShortUrl(ctx context.Context, shortUrl string) (string, error) {
    now := time.Now().Unix()
    
    // 层级1：本地LRU缓存（无锁）
    if val, ok := c.lru.Get(shortUrl); ok {
        if item, ok := val.(lruItem); ok && item.expiredAt >= now {
            return item.originUrl, nil
        }
    }
    
    // 层级2：Redis缓存（无合并）
    originUrl, err := c.cache.Get(ctx, shortUrl)
    if err == nil {
        // 异步刷新Redis TTL
        go c.asyncRefreshRedis(shortUrl)
        // 更新本地缓存
        c.updateLRU(shortUrl, originUrl)
        return originUrl, nil
    }
    
    // 层级3：数据库查询（带合并）
    result, err, _ := c.requestGroup.Do("db_query_"+shortUrl, func() (interface{}, error) {
        su, err := c.dao.FindByShortUrlWithExpired(ctx, shortUrl, now)
        if err != nil {
            return "", err
        }
        
        // 异步回填缓存
        go c.asyncFillCache(shortUrl, su.OriginUrl)
        
        return su.OriginUrl, nil
    })
    
    if err != nil {
        return "", err
    }
    return result.(string), nil
}
```

本地缓存大于网络缓存，所以采用异步删除网络缓存

```go
func (c *CachedShortUrlRepository) DeleteShortUrlByShortUrl(ctx context.Context, shortUrl string) error {
    err := c.dao.DeleteByShortUrl(ctx, shortUrl)
    if err == nil {
       go func() {
          newCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
          defer cancel()

          // 异步删除 redis 缓存
          if err = c.cache.Del(newCtx, shortUrl); err != nil {
             c.l.Error("failed to delete redis cache",
                logger.Error(err),
                logger.String("short_url", shortUrl),
             )
          }
       }()
       // 同步删除本地 lru 缓存
       c.lru.Remove(shortUrl)
    }
    return err
}

func (c *CachedShortUrlRepository) CleanExpired(ctx context.Context, now int64) error {
	deleteList, err := c.dao.DeleteExpiredList(ctx, now)
	if err == nil {
		newCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		go func() {
			defer cancel()
			for _, shortUrl := range deleteList {
				// 异步删除 redis 缓存
				if err = c.cache.Del(newCtx, shortUrl); err != nil {
					c.l.Error("failed to delete redis cache",
						logger.Error(err),
						logger.String("short_url", shortUrl),
					)
				}
				// 异步删除本地 lru 缓存
				c.lru.Remove(shortUrl)
			}
		}()
	}
	return err
}

```

键的设计：（准确的tag可以加速查询和减少隐患）

```go
type ShortUrl struct {
    ShortUrl  string `gorm:"type:char(7) CHARACTER SET ascii COLLATE ascii_bin;not null;primaryKey;column:short_url"`
    OriginUrl string `gorm:"type:varchar(200) CHARACTER SET ascii COLLATE ascii_bin;not null;default '';uniqueIndex:uk_origin_url"`
    ExpiredAt int64  `gorm:"type:bigint;default '-1':index:idx_expired_at"`
}
```

dao层的优化设计：

插入的设计：

插入流程图

![image-20250814231437292](C:\Users\linweihao\AppData\Roaming\Typora\typora-user-images\image-20250814231437292.png)

###  体系

|  **层级**  |    技术    |   优化目标   |
| :--------: | :--------: | :----------: |
|  内存缓冲  | 环形缓冲区 |    写合并    |
|  批量处理  | 对象池复用 | 减少内存分配 |
| 异步持久化 |   工作池   |  解耦主流程  |
|  数据插入  |    分表    | 单表写入瓶颈 |

代码：

dao结构体实现

前置知识：[](https://juejin.cn/post/7220043684292411451)

```go
type GormShortUrlDAO struct {
    db            *gorm.DB
    l             logger.Logger
    buffer        []ShortUrl      // 环形缓冲区
    bufferSize    int             // 缓冲区大小
    readPos       int             // 读取位置
    writePos      int             // 写入位置
    batchSize     int             // 批量大小
    flushInterval time.Duration   // 刷新间隔
    wg            sync.WaitGroup  // 用于等待worker完成
    stopChan      chan struct{}   // 用于停止worker
    flushPool     sync.Pool       // 用于批量处理的slice池
    flushChan     chan []ShortUrl // 用于异步刷新
    mu            sync.Mutex      // 保护缓冲区访问
}
```

```go
func NewGormShortUrlDAO(db *gorm.DB, l logger.Logger) ShortUrlDAO {
    dao := &GormShortUrlDAO{
       db:            db,
       l:             l,
       buffer:        make([]ShortUrl, 2000), // 双倍大小以提供缓冲
       bufferSize:    2000,
       batchSize:     1000,
       flushInterval: 50 * time.Millisecond,
       stopChan:      make(chan struct{}),
       flushChan:     make(chan []ShortUrl, 10), // 10个并发刷新
    }

    // 初始化slice池（复用容器，减少gc压力）
    dao.flushPool.New = func() interface{} {
       return make([]ShortUrl, 0, dao.batchSize)
    }
	
    // 双重保障：WaitGroup控制完成 + Context控制超时
    //先停止接受新任务，再处理剩余任务
    //多级WaitGroup嵌套
    dao.wg.Add(1)
    //这个wg的waith呢
    
    
    //启动缓冲区优雅关机和定时刷新（1）
    go dao.batchWorker()

    // 启动异步刷新worker
    for i := 0; i < 5; i++ {
       go func() {
           //从通道取出对象池
          for batch := range dao.flushChan {
             ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
             //启动worker（2）
             dao.flushBatch(ctx, batch)
             cancel()
          }
       }()
    }
	
    return dao
}
//（1）
func (g *GormShortUrlDAO) batchWorker() {
	defer g.wg.Done()

	ticker := time.NewTicker(g.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-g.stopChan:
			// 处理剩余请求
			g.flushRemaining()
			return
		case <-ticker.C:
			g.flushIfNeeded()
		}
	}
}
//（2）worker

func (g *GormShortUrlDAO) flushBatch(ctx context.Context, batch []ShortUrl) {
	defer g.flushPool.Put(batch)

	// 按表名分组
	groups := make(map[string][]ShortUrl)
	for _, su := range batch {
		table := g.tableName(su.ShortUrl)
		groups[table] = append(groups[table], su)
	}

	// 使用worker池处理每个表
func (g *GormShortUrlDAO) flushBatch(ctx context.Context, batch []ShortUrl) {
    // 1. 对象池归还
    defer g.flushPool.Put(batch)

    // 2. 按表名分组
    groups := make(map[string][]ShortUrl)
    for _, su := range batch {
        table := g.tableName(su.ShortUrl)
        groups[table] = append(groups[table], su)
    }

    // 3. 并发处理各表
    var wg sync.WaitGroup
    wg.Add(len(groups)) // 关键点：精准计数

    for table, sus := range groups {
        go func(table string, sus []ShortUrl) {
            defer wg.Done() // 任务完成标记
            
            // 4. 事务处理
            g.db.WithContext(ctx).Table(table).Transaction(func(tx *gorm.DB) error {
                // 5. 批量插入（带冲突处理）
                result := tx.Clauses(clause.OnConflict{
                    Columns:   []clause.Column{{Name: "short_url"}},//空映射含义：冲突时不更新任何字段
                    DoUpdates: clause.Assignments(map[string]any{}),
                }).Create(&sus)

                // 6. 冲突检测
                if result.RowsAffected != int64(len(sus)) {
                    // 7. 冲突处理(红灯区域，类似于c++，unique函数)
                    for i := result.RowsAffected; i < int64(len(sus)); i++ {
                        var existing ShortUrl
                        if err := tx.Where("short_url = ?", sus[i].ShortUrl).First(&existing).Error; err != nil {
                            // 8. 记录错误
                            g.l.Error("batch insert conflict check failed",
                                logger.Error(err),
                                logger.String("short_url", sus[i].ShortUrl))
                            continue
                        }
                        // 9. 主键冲突警告
                        if existing.OriginUrl != sus[i].OriginUrl {
                            g.l.Warn("primary key conflict detected",
                                logger.String("short_url", sus[i].ShortUrl),
                                logger.String("existing_origin_url", existing.OriginUrl),
                                logger.String("new_origin_url", sus[i].OriginUrl))
                        }
                    }
                }
                return nil
            })
        }(table, sus) // 注意：闭包参数传递
    }

    // 10. 等待所有表完成
    wg.Wait()
}
  //优雅关机
func (g *GormShortUrlDAO) Close() error {
	close(g.stopChan)
	g.wg.Wait()
	return nil
}
//插入
func (g *GormShortUrlDAO) Insert(ctx context.Context, su ShortUrl) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	nextPos := (g.writePos + 1) % g.bufferSize
	if nextPos == g.readPos {
		return errors.New("buffer full")
	}

	g.buffer[g.writePos] = su
	g.writePos = nextPos

	// 如果达到批量大小，立即刷新
	if (g.writePos-g.readPos+g.bufferSize)%g.bufferSize >= g.batchSize {
		g.flushIfNeeded()
	}

	return nil
}    
//删除部分，为防止压垮数据库，采用阻塞等待
    func (g *GormShortUrlDAO) DeleteExpiredList(ctx context.Context, now int64) ([]string, error) {
	var (
		retList []string
		group   errgroup.Group
		lock    sync.Mutex
	)
      //  errgroup.Group是 Go 语言中一个非常有用的并发原语，它来自 golang.org/x/sync/errgroup包。它主要用于管理一组 goroutine，并提供了以下关键特性：

//并发执行：可以启动多个 goroutine 来执行任务。
//错误传播：如果任何一个 goroutine 返回错误，整个组会取消所有其他 goroutine（通过上下文取消）。
//等待所有完成：可以等待所有 goroutine 完成，并返回第一个发生的错误（如果有的话）。
	for i := 0; i < 62; i++ {
		group.Go(func() error {
			tableName := "short_url_" + string(generator.BASE62CHARSET[i])
			for {
				var ret []string
				// 查询可删除列表
				err := g.db.WithContext(ctx).Table(tableName).Select("short_url").
					Where("expired_at < ?", now).Order("expired_at ASC").Limit(100).
					Find(&ret).Error
				if err != nil {
					return err
				}
				if len(ret) == 0 {
					break // 无更多数据可删除
				}
				err = g.db.WithContext(ctx).Table(tableName).Where("short_url IN ?", ret).Delete(&ShortUrl{}).Error
				if err != nil {
					return err
				}

				lock.Lock()
				retList = append(retList, ret...)
				lock.Unlock()

				time.Sleep(100 * time.Millisecond) // 避免高频操作压垮数据库
			}
			return nil
		})
	}
	return retList, group.Wait()
}
    
//关于分表和不分表操作：前者会返回找到的第一个，后者会全部找到
func (g *GormShortUrlDAO) FindExpiredList(ctx context.Context, now int64) ([]ShortUrl, error) {
	var (
		sus  []ShortUrl
		wg   sync.WaitGroup
		lock sync.Mutex
	)
	newCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	wg.Add(62)
	for i := 0; i < 62; i++ {
		go func(internalCtx context.Context, suffix string) {
			defer wg.Done()
			select {
			case <-internalCtx.Done():
				return
			default:
				var internalSus []ShortUrl
				if err := g.db.WithContext(internalCtx).
					Table(g.tableName(suffix)).
					Where("expired_at <=?", now).
					Find(&internalSus).Error; err == nil {
					lock.Lock()
					sus = append(sus, internalSus...)
					lock.Unlock()
					cancel()
				}
			}
		}(newCtx, string(generator.BASE62CHARSET[i]))
	}
	wg.Wait()
	if len(sus) == 0 {
		return nil, ErrDataNotFound
	}
	return sus, nil
}

func (g *GormShortUrlDAO) FindExpiredListV1(ctx context.Context, now int64) ([]ShortUrl, error) {
	var (
		sus  []ShortUrl
		lock sync.Mutex
	)
	g.executeUnshardedQuery(ctx, func(iCtx context.Context, suffix string, db *gorm.DB) error {
		var internalSus []ShortUrl
		err := db.WithContext(iCtx).
			Table(g.tableName(suffix)).
			Where("expired_at <=?", now).
			Find(&internalSus).Error
		if err != nil {
			g.l.Error("FindExpiredListV1 failed",
				logger.Error(err),
				logger.String("suffix", suffix),
				logger.Int64("expired_at", now),
			)
			return err
		}
		lock.Lock()
		sus = append(sus, internalSus...)
		lock.Unlock()
		return nil
	})
	if len(sus) == 0 {
		return nil, ErrDataNotFound
	}
	return sus, nil
}

// 批量执行不分表操作的抽象方法
func (g *GormShortUrlDAO) executeUnshardedQuery(ctx context.Context, fn func(iCtx context.Context, suffix string, db *gorm.DB) error) {
	var wg sync.WaitGroup
	newCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	wg.Add(62)
	for i := 0; i < 62; i++ {
		go func(internalCtx context.Context, suffix string) {
			defer wg.Done()
			select {
			case <-internalCtx.Done():
				return
			default:
				if err := fn(internalCtx, suffix, g.db); err == nil {
					cancel()
				}
			}
		}(newCtx, string(generator.BASE62CHARSET[i]))
	}
	wg.Wait()
}

```





待：

```go
func (g *GormShortUrlDAO) Transaction(ctx context.Context, fc func(tx *gorm.DB) error, opts ...*sql.TxOptions) error {
    return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
       return fc(tx)
    }, opts...)
}

func (g *GormShortUrlDAO) WithTransaction(ctx context.Context, fc func(txDAO ShortUrlDAO) error, opts ...*sql.TxOptions) error {
    return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
       txDAO := &GormShortUrlDAO{db: tx}
       return fc(txDAO)
    }, opts...)
}
```



14 数据导出：采用原生库，体量小 [原生sql库](https://www.liwenzhou.com/posts/Go/mysql/)

```go
func main() {
    // 数据库连接
    db, err := sql.Open("mysql", "root:123456@tcp(127.0.0.1:3306)/short_url")
    if err != nil {
       panic(err)
    }
    defer db.Close()

    // 创建CSV文件
    file, err := os.Create("short_url_all.csv")
    if err != nil {
       panic(err)
    }
    defer file.Close()

    writer := csv.NewWriter(file)
    defer writer.Flush()

    // 写入CSV头
    writer.Write([]string{"short_url"})

    // 获取所有short_url_前缀的表
    rows, err := db.Query("SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME LIKE 'short_url_%'")
    if err != nil {
       panic(err)
    }
    defer rows.Close()

    // 遍历每个表
    for rows.Next() {
       var tableName string
       if err := rows.Scan(&tableName); err != nil {
          panic(err)
       }

       // 查询该表的short_url
       query := fmt.Sprintf("SELECT short_url FROM %s", tableName)
       dataRows, err := db.Query(query)
       if err != nil {
          panic(err)
       }
       defer dataRows.Close()

       // 写入CSV
       for dataRows.Next() {
          var shortUrl string
          if err := dataRows.Scan(&shortUrl); err != nil {
             panic(err)
          }
          writer.Write([]string{shortUrl})
       }
    }

    fmt.Println("All short_urls exported to short_url_all.csv")
}
```

web和rpc调用的逻辑（第一次学，记一下下）：

![image-20250815194746703](C:\Users\linweihao\AppData\Roaming\Typora\typora-user-images\image-20250815194746703.png)

grpc_config

```yaml
grpc: 
  server:
    port: 0  # 填 0 随机分配空闲端口
    etcdTTL: 60 #服务注册时获得60秒租约,需要心跳续约
    etcdAddr: "127.0.0.1:2379" #指定服务注册的etcd地址

etcd:  #访问etcd的地址，所有组件访问etcd的地址
  endpoints:
    - "127.0.0.1:2379"
```

web_config

```yaml
etcd: 
  addrs:	#Web服务查询服务地址的地址
    - "localhost:2379"

grpc:
  client:
    shortUrl:	#服务在etcd中的注册路径
      target: "etcd:///service/short_url" 
      Secure: false #禁用TLS（仅限开发环境）
```

web利用原生的重定向：

```go
func (h *ServerHandler) Redirect(ctx *gin.Context) {
	shortUrl := ctx.Param("short_url")
	if ok := generator.CheckShortUrl(shortUrl, h.weights); !ok {
		ctx.JSON(404, gin.H{"error": "Short URL not found"})
		return
	}

	result, err, _ := h.requestGroup.Do(shortUrl, func() (interface{}, error) {
		resp, err := h.svc.GetOriginUrl(ctx, &short_url_v1.GetOriginUrlRequest{
			ShortUrl: shortUrl,
		})
		if err != nil {
			return "", err
		}
		return resp.GetOriginUrl(), nil
	})
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	ctx.Redirect(301, result.(string))
}

```



# 内存监控测试，mem[包](https://segmentfault.com/a/1190000022281174)

```go
package test

import (
	"testing"

	"github.com/shirou/gopsutil/v3/mem" // 导入系统内存信息库
)

// TestMem_VirtualMemory 测试函数 - 获取并分析系统虚拟内存信息
// 该测试用于监控系统内存使用情况，为容量规划和性能优化提供数据支持
func TestMem_VirtualMemory(t *testing.T) {
	// 获取系统虚拟内存信息
	// VirtualMemory() 返回包含内存使用详细数据的结构体
	v, err := mem.VirtualMemory()
	if err != nil {
		// 如果获取内存信息失败，终止测试并标记为失败
		t.Fatal(err)
	}

	// 输出完整内存信息结构体（调试用）
	t.Log("v:", v)
	
	// 输出内存使用百分比 - 核心监控指标
	// 用于判断系统内存压力（>80%需告警）
	t.Log("v.UsedPercent:", v.UsedPercent, "%")
	
	// 输出总内存大小（字节）
	t.Log("v.Total(B):", v.Total)
	
	// 输出总内存大小（GB） - 更易读的格式
	// 转换公式：字节 → GB（除以1024^3）
	t.Log("v.Total(GB):", float64(v.Total)/1024/1024/1024)
	
	// 输出空闲内存量（字节）
	t.Log("v.Free:", v.Free)
	
	// 输出已用内存量（字节）
	t.Log("v.Used:", v.Used)
	
	// 输出共享内存量（字节）
	// 共享内存：多个进程可同时访问的内存区域
	t.Log("v.Shared:", v.Shared)
	
	// 输出缓冲区内存量（字节）
	// 缓冲区：内核用于块设备I/O的临时存储
	t.Log("v.Buffers:", v.Buffers)
	
	// 输出缓存内存量（字节）
	// 缓存：从磁盘读取的文件页缓存
	t.Log("v.Cached:", v.Cached)
	
	// 输出活跃内存量（字节）
	// 活跃内存：最近被访问过的内存页
	t.Log("v.Active:", v.Active)
	
	// 输出非活跃内存量（字节）
	// 非活跃内存：最近未被访问的内存页（可回收）
	t.Log("v.Inactive:", v.Inactive)

	// 计算并输出特定内存分配策略值
	// 公式解释：(总内存 × 3%) ÷ 256
	// 用途：可能是为内存池分配计算块数量
	// 示例：16GB内存 → (16×0.03)/0.25 = 1.92 → 2个256MB块
	t.Log("calculate:", int(float64(v.Total)/100 * 3/256))
}
```

compose文件

```yaml
version: "3"

services:
  mysql8:
    image: mysql:8.0
    restart: always
    command:
      - --default-authentication-plugin=mysql_native_password
      - --binlog-format=ROW
      - --server-id=1
      - --max_connections=500
    environment:
      MYSQL_ROOT_PASSWORD: "123456"
      MYSQL_INNODB_BUFFER_POOL_SIZE: "1G"
      MYSQL_INNODB_LOG_BUFFER_SIZE: "64M"
      MYSQL_BULK_INSERT_BUFFER_SIZE: "256M" # 批量插入专用内存
      MYSQL_INNODB_FLUSH_LOG_AT_TRX_COMMIT: "0" # 事务日志刷新策略(0=性能优先，可能会丢失1秒日志)
      MYSQL_INNODB_AUTOCOMMIT: "0" # 禁用自动提交提升批量效率
    volumes:
      - ./scripts/mysql/:/docker-entrypoint-initdb.d/
    ports:
      - 3306:3306
    
  redis:
    image: "bitnami/redis:latest"
    restart: always
    environment:
      - ALLOW_EMPTY_PASSWORD=yes
    ports:
      - "6379:6379"

  etcd:
    image: "bitnami/etcd:latest"
    restart: always
    volumes:
      - "./scripts/etcd_data:/etcd-data"
    environment:
      - ALLOW_NONE_AUTHENTICATION=yes
    ports:
      - "2379:2379"
```

**只执行一次的限制**：

`/docker-entrypoint-initdb.d/`只在**第一次创建容器时执行**

如果容器已经创建过（即使停止/重启），脚本不会再次执行