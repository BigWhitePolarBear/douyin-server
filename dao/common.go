package dao

import (
	jsoniter "github.com/json-iterator/go"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

type Comment struct {
	Id         int64  `json:"id,omitempty" gorm:"primaryKey"`
	UserId     int64  `json:"-"`
	VideoId    int64  `json:"video_id"`
	CreateDate string `json:"create_date,omitempty" gorm:"type:varchar(10);index"`
	Content    string `json:"content,omitempty" gorm:"type:text"`

	User User `json:"user"`
}

type Favorite struct {
	UserId  int64 `json:"-" gorm:"primaryKey;autoIncrement:false"`
	VideoId int64 `json:"-" gorm:"primaryKey;autoIncrement:false"`
}
