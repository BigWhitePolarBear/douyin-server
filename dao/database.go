package dao

import (
	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"time"
)

var (
	DB *gorm.DB

	LoginCache     *cache.Cache
	UserCache      *cache.Cache
	RdbToken       *redis.Client
	RdbFollow      *redis.Client // 存放关注列表
	RdbFans        *redis.Client // 存放粉丝列表
	VideoCache     *cache.Cache
	VideoListCache *cache.Cache
)

// Redis数据库编号
const (
	numTokenDB = iota
	numLoginCacheDB
	numUserCacheDB
	numFollowListDB
	numFollowerListDB
	numVideoCacheDB
	numVideoListCacheDB
)

func InitDB() {
	var err error
	dsn := "douyin_server:zomFinEFjpv5Jj7oPw3hsoDcU51M865Y@tcp(localhost:23306)/" +
		"douyin?charset=utf8mb4&interpolateParams=true&parseTime=True&loc=Local"
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		panic(err)
	}

	err = DB.AutoMigrate(&User{}, &Video{}, &Favorite{}, &Comment{})
	log.Println(err)

	RdbToken = redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName:    "mymaster",
		SentinelAddrs: []string{":17000", ":17001", ":17002"},
		DB:            numTokenDB,
		Password:      "YvANrvTJLn2cm3u5vvzKc62uFBBiRanj",
	})

	LoginCache = cache.New(&cache.Options{
		Redis: redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    "mymaster",
			SentinelAddrs: []string{":17000", ":17001", ":17002"},
			DB:            numLoginCacheDB,
			Password:      "YvANrvTJLn2cm3u5vvzKc62uFBBiRanj",
		}),
		LocalCache: cache.NewTinyLFU(1000, time.Minute),
	})

	UserCache = cache.New(&cache.Options{
		Redis: redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    "mymaster",
			SentinelAddrs: []string{":17000", ":17001", ":17002"},
			DB:            numUserCacheDB,
			Password:      "YvANrvTJLn2cm3u5vvzKc62uFBBiRanj",
		}),
		LocalCache: cache.NewTinyLFU(1000, time.Minute),
	})

	// 关注列表数据库
	RdbFollow = redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName:    "mymaster",
		SentinelAddrs: []string{":17000", ":17001", ":17002"},
		DB:            numFollowListDB,
		Password:      "YvANrvTJLn2cm3u5vvzKc62uFBBiRanj",
	})
	// 粉丝列表数据库
	RdbFans = redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName:    "mymaster",
		SentinelAddrs: []string{":17000", ":17001", ":17002"},
		DB:            numFollowerListDB,
		Password:      "YvANrvTJLn2cm3u5vvzKc62uFBBiRanj",
	})

	VideoCache = cache.New(&cache.Options{
		Redis: redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    "mymaster",
			SentinelAddrs: []string{":17000", ":17001", ":17002"},
			DB:            numVideoCacheDB,
			Password:      "YvANrvTJLn2cm3u5vvzKc62uFBBiRanj",
		}),
		LocalCache: cache.NewTinyLFU(1000, time.Minute),
	})

	VideoListCache = cache.New(&cache.Options{
		Redis: redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    "mymaster",
			SentinelAddrs: []string{":17000", ":17001", ":17002"},
			DB:            numVideoListCacheDB,
			Password:      "YvANrvTJLn2cm3u5vvzKc62uFBBiRanj",
		}),
		LocalCache: cache.NewTinyLFU(1000, time.Minute),
	})
}
