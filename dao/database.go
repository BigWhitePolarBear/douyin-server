package dao

import (
	"context"
	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"time"
)

var (
	DB *gorm.DB

	RDB        *redis.Client
	Ctx        context.Context
	LoginCache *cache.Cache
)

// Redis数据库编号
const (
	numTokenDB = iota
	numLoginCacheDB
)

func InitDB() {
	var err error
	dsn := "root:1234@tcp(127.0.0.1:3306)/" +
		"douyin?charset=utf8&interpolateParams=true&parseTime=True&loc=Local"
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		panic(err)
	}

	err = DB.AutoMigrate(&User{}, &Video{}, &Favorite{})
	log.Println(err)

	RDB = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       numTokenDB,
	})
	Ctx = context.Background()

	LoginCache = cache.New(&cache.Options{
		Redis: redis.NewClient(&redis.Options{
			Addr:     "127.0.0.1:6379",
			Password: "",
			DB:       numLoginCacheDB,
		}),
		LocalCache: cache.NewTinyLFU(1000, time.Minute),
	})
}
