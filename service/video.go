package service

import (
	"context"
	"douyin-server/dao"
	"golang.org/x/sync/singleflight"
	"log"
	"strconv"
	"time"
)

var (
	videoGroup singleflight.Group
)

type Video struct {
	IsFavorite    bool   `json:"is_favorite,omitempty"`
	Id            int64  `json:"id,omitempty"`
	FavoriteCount int64  `json:"favorite_count,omitempty"`
	CommentCount  int64  `json:"comment_count,omitempty"`
	Title         string `json:"title,omitempty"`
	PlayUrl       string `json:"play_url,omitempty"`
	CoverUrl      string `json:"cover_url,omitempty"`

	Author dao.User `json:"author"`
}

func getVideo(id int64) (video dao.Video) {
	sID := strconv.FormatInt(id, 10)

	var jsonVideo []byte
	err := dao.VideoCache.Get(context.Background(), sID, &jsonVideo)
	if err == nil {
		err = json.Unmarshal(jsonVideo, &video)
		if err != nil {
			log.Println("视频元信息解码错误：", err)
			return
		}
	} else {
		// 缓存未命中
		_video, _err, _ := videoGroup.Do(sID, func() (interface{}, error) {
			go func() {
				time.Sleep(200 * time.Millisecond)
				videoGroup.Forget(sID)
			}()

			_video := dao.Video{}
			err = dao.DB.Model(&dao.Video{}).Where("id = ?", id).Find(&_video).Error
			return _video, err
		})
		err = _err
		if err != nil {
			log.Println("从数据库中读取视频元信息失败：", err)
			return
		}
		video = _video.(dao.Video)
	}

	video.Author, err = UserInfo(video.AuthorId)
	if err != nil {
		log.Println("获取视频作者信息失败：", err)
	}

	return
}
