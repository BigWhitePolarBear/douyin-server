package service

import (
	"context"
	"douyin-server/dao"
	"github.com/go-redis/cache/v8"
	"golang.org/x/sync/singleflight"
	"log"
	"strconv"
	"time"
)

var (
	videoListGroup singleflight.Group
)

// Feed 选择发布时间在latestTime之前的视频，时间戳每10秒一个点。
func Feed(id int64, latestTime int64) (videoList []Video, nextTime int64) {
	videoIdList := make([]int64, 0)
	timeForRetrieve := strconv.FormatInt(latestTime/10000, 10)
	err := dao.VideoListCache.Get(context.Background(), timeForRetrieve, &videoIdList)
	if err != nil {
		// 视频列表缓存未命中
		_videoIdList, _, _ := videoListGroup.Do(timeForRetrieve, func() (interface{}, error) {
			go func() {
				time.Sleep(200 * time.Millisecond)
				videoListGroup.Forget(timeForRetrieve)
			}()

			timeStamp := time.UnixMilli(latestTime)
			_videoIdList := make([]int64, 0)
			dao.DB.Model(&dao.Video{}).Preload("Author").Where("created_at <= ?", timeStamp).
				Order("created_at desc").Select("id").Limit(10).Find(&_videoIdList)
			err = dao.VideoListCache.Set(&cache.Item{
				Key:   timeForRetrieve,
				Value: _videoIdList,
				TTL:   10 * time.Second,
			})
			if err != nil {
				log.Println("视频列表缓存时错误：", err)
			}
			return _videoIdList, nil
		})
		videoIdList = _videoIdList.([]int64)
	}

	// 没有视频，返回空
	if len(videoIdList) == 0 {
		return
	}

	// 根据视频id获取视频元信息
	_videoList := make([]dao.Video, len(videoIdList))
	for i := range videoIdList {
		_videoList[i] = getVideo(videoIdList[i])
	}

	// 返回这次视频最近的投稿时间-1，下次即可获取比这次视频旧的视频
	nextTime = _videoList[len(_videoList)-1].CreatedAt.UnixMilli() - 1

	videoList = make([]Video, len(_videoList))

	for i := range _videoList {
		videoList[i].Id, videoList[i].Title = _videoList[i].Id, _videoList[i].Title
		videoList[i].FavoriteCount, videoList[i].CommentCount = _videoList[i].FavoriteCount, _videoList[i].CommentCount
		videoList[i].PlayUrl, videoList[i].CoverUrl = _videoList[i].PlayUrl, _videoList[i].CoverUrl

		// 此处错误可忽略
		author, _ := UserInfo(_videoList[i].AuthorId)
		videoList[i].Author = author
		// 说明当前用户已登录
		if id != 0 {
			// 点赞，该错误可忽略
			rows, _ := dao.DB.Model(&dao.Favorite{}).Where("user_id = ? AND video_id = ?", id, videoList[i].Id).Rows()
			if rows.Next() {
				videoList[i].IsFavorite = true
			}

			// 关注
			followed := dao.RdbFollow.HExists(context.Background(), strconv.FormatInt(id, 10),
				strconv.FormatInt(videoList[i].Author.Id, 10)).Val()
			if followed {
				videoList[i].Author.IsFollow = true
			}
		}
	}
	return
}
