package dao

import (
	"context"
	"github.com/go-redis/cache/v8"
	"gorm.io/gorm"
	"log"
	"strconv"
	"time"
)

type Video struct {
	Id            int64     `json:"id,omitempty" gorm:"primaryKey"`
	AuthorId      int64     `json:"author_id"`
	FavoriteCount int64     `json:"favorite_count,omitempty"`
	CommentCount  int64     `json:"comment_count,omitempty"`
	Title         string    `json:"title,omitempty" gorm:"type:varchar(100)"`
	PlayUrl       string    `json:"play_url,omitempty" gorm:"type:varchar(100)"`
	CoverUrl      string    `json:"cover_url,omitempty" gorm:"type:varchar(100)"`
	CreatedAt     time.Time `json:"-" gorm:"index:,sort:desc"`

	Author User `json:"-"`
}

// BeforeSave 进行延迟双删第一删
func (v *Video) BeforeSave(tx *gorm.DB) (err error) {
	v.deleteFromCache()
	return nil
}

// AfterSave 进行延迟双删第二删
func (v *Video) AfterSave(tx *gorm.DB) (err error) {
	go func() {
		time.Sleep(200 * time.Millisecond)
		v.deleteFromCache()
	}()
	return nil
}

// BeforeUpdate 进行延迟双删第一删
func (v *Video) BeforeUpdate(tx *gorm.DB) (err error) {
	v.deleteFromCache()
	return nil
}

// AfterUpdate 进行延迟双删第二删
func (v *Video) AfterUpdate(tx *gorm.DB) (err error) {
	go func() {
		time.Sleep(200 * time.Millisecond)
		v.deleteFromCache()
	}()
	return nil
}

// AfterFind 查询完后进行写缓存
func (v *Video) AfterFind(tx *gorm.DB) (err error) {
	v.saveIntoCache()

	// 无论是否成功写缓存，继续完成事务
	return nil
}

// AfterCreate 创建完后进行写缓存
func (v *Video) AfterCreate(tx *gorm.DB) (err error) {
	v.saveIntoCache()

	// 无论是否成功写缓存，继续完成事务
	return nil
}

func (v *Video) saveIntoCache() {
	jsonV, err := json.Marshal(*v)
	if err != nil {
		log.Println("视频信息编码失败:", err)
		return
	}

	err = VideoCache.Set(&cache.Item{
		Key:   strconv.FormatInt(v.Id, 10),
		Value: jsonV,
		TTL:   time.Minute,
	})
	if err != nil {
		log.Println("视频信息缓存失败:", err)
	}
}

func (v *Video) deleteFromCache() {
	_ = VideoCache.Delete(context.Background(), strconv.FormatInt(v.Id, 10))
}
