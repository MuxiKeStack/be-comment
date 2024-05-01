package domain

import (
	commentv1 "github.com/MuxiKeStack/be-api/gen/proto/comment/v1"
	"time"
)

type Comment struct {
	Id int64 `json:"id"`
	// 评论者
	Commentator User `json:"commentator"`
	// 评论对象
	// 数据里面
	Biz   commentv1.Biz `json:"biz"`
	BizId int64         `json:"bizid"`
	// 评论对象
	Content string `json:"content"`
	// 根评论
	RootComment *Comment `json:"rootComment"`
	// 父评论
	ParentComment *Comment  `json:"parentComment"`
	ReplyToUid    int64     `json:"reply_to_uid"`
	Children      []Comment `json:"children"`
	CTime         time.Time `json:"ctime"`
	UTime         time.Time `json:"utime"`
}

type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
