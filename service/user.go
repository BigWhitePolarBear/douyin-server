package service

import (
	"bytes"
	"context"
	"douyin-server/dao"
	"errors"
	"fmt"
	"github.com/bits-and-blooms/bloom/v3"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

var (
	userNameFilter *bloom.BloomFilter
	userIdFilter   *bloom.BloomFilter
)

// InitUser 等dao包初始化完才能初始化
func InitUser() {
	// 支持10000000个用户
	userIdFilter = bloom.NewWithEstimates(1e7, 0.01)
	userNameFilter = bloom.NewWithEstimates(1e7, 0.01)

	rows, err := dao.DB.Model(dao.User{}).Select("id", "name").Rows()
	if err != nil {
		panic(err)
	}
	defer func() {
		err = rows.Close()
		if err != nil {
			panic(err)
		}
	}()

	// 将数据库中所有用户名存在布隆过滤器中
	var id, name string
	for rows.Next() {
		err = rows.Scan(&id, &name)
		if err != nil {
			log.Println("读取用户名到布隆过滤器时发生错误：", err)
		}
		userIdFilter.AddString(id)
		userNameFilter.AddString(name)
	}
}

// LoginLimit 中间件服务，限制注册登录操作过于频繁。
func LoginLimit(ipAddress string) bool {
	// 错误可忽略
	times, _ := dao.RdbToken.Get(context.Background(), ipAddress).Int64()
	if times > 10 {
		return false
	} else {
		dao.RdbToken.Set(context.Background(), ipAddress, times+1, time.Minute)
	}
	return true
}

func Register(username, password string) (id int64, err error) {
	if len(username) > 32 {
		return 0, errors.New("用户名过长，不可超过32位")
	}
	if len(username) == 0 {
		return 0, errors.New("用户名不可为空")
	}
	if len(password) > 32 {
		return 0, errors.New("密码过长，不可超过32位")
	}

	user := dao.User{}
	dao.DB.Model(&dao.User{}).Where("name = ?", username).Find(&user)
	if user.Id != 0 {
		return 0, errors.New("用户已存在")
	}

	user.Name = username

	// 加密存储用户密码
	user.Salt = randSalt()
	buf := bytes.Buffer{}
	buf.WriteString(username)
	buf.WriteString(password)
	buf.WriteString(user.Salt)
	pwd, err := bcrypt.GenerateFromPassword(buf.Bytes(), bcrypt.MinCost)
	if err != nil {
		return 0, err
	}
	user.Pwd = string(pwd)

	dao.DB.Model(&dao.User{}).Create(&user)

	// 布隆过滤器中加入新用户
	userIdFilter.AddString(strconv.FormatInt(user.Id, 10))
	userNameFilter.AddString(username)

	return user.Id, nil
}

func Login(username, password string) (id int64, err error) {
	// 先查布隆过滤器，不存在直接返回错误，降低数据库的压力
	if !userNameFilter.TestString(username) {
		return 0, errors.New("用户名或密码错误")
	}

	user := dao.User{}

	// 再查缓存
	cacheMissed := false
	var buf []byte
	err = dao.LoginCache.Get(context.Background(), username, &buf)
	if err == nil {
		err = json.Unmarshal(buf, &user)
		if err != nil {
			cacheMissed = true
		}
	} else {
		cacheMissed = true
	}

	//缓存未命中，查数据库
	if cacheMissed {
		// 上下文中注明本次要写入登录缓存
		dao.DB.Model(&dao.User{}).Set("login", struct{}{}).Where("name = ?", username).Find(&user)
	}

	// 检验密码
	if err = bcrypt.CompareHashAndPassword([]byte(user.Pwd), []byte(username+password+user.Salt)); err != nil {
		return 0, errors.New("用户名或密码错误")
	}
	return user.Id, nil
}

// UserInfo 根据用户id获取用户除敏感字段外的完整信息
func UserInfo(id int64) (user dao.User, err error) {
	// 先查布隆过滤器，过滤不存在的id
	if !userIdFilter.TestString(strconv.FormatInt(id, 10)) {
		return user, errors.New("用户不存在")
	}

	// 分别获取各个字段
	val, err := UserInfoByField(id, "Name")
	if err != nil {
		return
	}
	user.Name = val

	val, err = UserInfoByField(id, "TotalFavorited")
	if err != nil {
		return
	}
	user.TotalFavorited, err = strconv.ParseInt(val, 10, 64)
	if err != nil {
		return
	}

	val, err = UserInfoByField(id, "FavoriteCount")
	if err != nil {
		return
	}
	user.FavoriteCount, err = strconv.ParseInt(val, 10, 64)
	if err != nil {
		return
	}

	userId := strconv.FormatInt(id, 10)
	user.FollowCount = dao.RdbFollow.HLen(context.Background(), userId).Val() // 用户的关注总数
	user.FollowerCount = dao.RdbFans.HLen(context.Background(), userId).Val() // 用户的粉丝总数

	user.Id = id

	return user, nil
}

// UserInfoByField 先查缓存再查数据库获得用户指定字段数据
func UserInfoByField(id int64, field string) (val string, err error) {
	// 先查布隆过滤器，过滤不存在的id
	if !userIdFilter.TestString(strconv.FormatInt(id, 10)) {
		return "", errors.New("用户不存在")
	}

	if field == "Name" {
		err = dao.UserCache.Get(context.Background(), fmt.Sprintf("%d:Name", id), &val)
		if err != nil {
			err = dao.DB.Model(&dao.User{Id: id}).Where("id = ?", id).Select("name").Find(&val).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", errors.New("用户不存在")
			}
		}
		return
	}

	if field == "TotalFavorited" {
		err = dao.UserCache.Get(context.Background(), fmt.Sprintf("%d:TotalFavorited", id), &val)
		if err != nil {
			err = dao.DB.Model(&dao.User{Id: id}).Where("id = ?", id).Select("total_favorited").Find(&val).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", errors.New("用户不存在")
			}
		}
		return
	}

	if field == "FavoriteCount" {
		err = dao.UserCache.Get(context.Background(), fmt.Sprintf("%d:FavoriteCount", id), &val)
		if err != nil {
			err = dao.DB.Model(&dao.User{Id: id}).Where("id = ?", id).Select("favorite_count").Find(&val).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", errors.New("用户不存在")
			}
		}
		return
	}

	return "", errors.New("UserInfoByField：field传入了非法参数")
}

// CreateToken 生成随机token，并存储到redis中，返回token
func CreateToken(id int64) (token int64) {
	// redis存储64位整数更节省空间
	token = int64(rand.Uint64())

	// 检测token有无冲突
	_, err := dao.RdbToken.Get(context.Background(), strconv.FormatInt(token, 10)).Result()
	for err == nil {
		token = int64(rand.Uint64())
		_, err = dao.RdbToken.Get(context.Background(), strconv.FormatInt(token, 10)).Result()
	}

	dao.RdbToken.Set(context.Background(), strconv.FormatInt(token, 10), id, 12*time.Hour)

	return
}

// 随机盐长度固定为4
func randSalt() string {
	buf := strings.Builder{}
	for i := 0; i < 4; i++ {
		// 如果写byte会无法兼容mysql编码
		buf.WriteRune(rune(rand.Intn(256)))
	}
	return buf.String()
}
