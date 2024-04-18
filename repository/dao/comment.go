package dao

import (
	"context"
	"database/sql"
	"errors"
	"gorm.io/gorm"
	"time"
)

type CommentDAO interface {
	FindByBiz(ctx context.Context, biz int32, bizId int64, curCommentId int64, limit int64) ([]Comment, error)
	FindRepliesByPid(ctx context.Context, pid int64, offset int, limit int) ([]Comment, error)
	Delete(ctx context.Context, commentId int64, uid int64) error
	GetCountByBiz(ctx context.Context, biz int32, bizId int64) (int64, error)
	FindRepliesByRid(ctx context.Context, rid int64, curCommentId int64, limit int64) ([]Comment, error)
	Insert(ctx context.Context, comment Comment) error
	FindById(ctx context.Context, commentId int64) (Comment, error)
}

type GORMCommentDAO struct {
	db *gorm.DB
}

func (dao *GORMCommentDAO) FindById(ctx context.Context, commentId int64) (Comment, error) {
	var c Comment
	err := dao.db.WithContext(ctx).
		Where("id = ?", commentId).
		First(&c).Error
	return c, err
}

func NewCommentDAO(db *gorm.DB) CommentDAO {
	return &GORMCommentDAO{
		db: db,
	}
}

// FindByBiz 先新后旧
func (dao *GORMCommentDAO) FindByBiz(ctx context.Context, biz int32, bizId int64, curCommentId int64, limit int64) ([]Comment, error) {
	var res []Comment
	err := dao.db.WithContext(ctx).
		Where("biz = ? AND biz_id = ? AND id < ? AND pid IS NULL", biz, bizId, curCommentId).
		Order("utime desc").
		Limit(int(limit)).
		Find(&res).Error
	return res, err
}

func (dao *GORMCommentDAO) Delete(ctx context.Context, commentId int64, uid int64) error {
	res := dao.db.WithContext(ctx).
		Where("id = ? and uid = ?", commentId, uid).
		Delete(&Comment{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("删除失败")
	}
	return nil
}

func (dao *GORMCommentDAO) GetCountByBiz(ctx context.Context, biz int32, bizId int64) (int64, error) {
	var count int64
	err := dao.db.WithContext(ctx).
		Model(&Comment{}).
		Where("biz = ? and biz_id = ?", biz, bizId).
		Count(&count).Error
	return count, err
}

// FindRepliesByRid 先旧后新
func (dao *GORMCommentDAO) FindRepliesByRid(ctx context.Context,
	rid int64, curCommentId int64, limit int64) ([]Comment, error) {
	var res []Comment
	err := dao.db.WithContext(ctx).
		Where("root_id = ? AND id > ?", rid, curCommentId).
		Order("id ASC").
		Limit(int(limit)).Find(&res).Error
	return res, err
}

// FindRepliesByPid 查找评论的直接评论
func (dao *GORMCommentDAO) FindRepliesByPid(ctx context.Context, pid int64, offset, limit int) ([]Comment, error) {
	var res []Comment
	err := dao.db.WithContext(ctx).Where("pid = ?", pid).
		Order("id DESC").
		Offset(offset).Limit(limit).Find(&res).Error
	return res, err
}

func (dao *GORMCommentDAO) Insert(ctx context.Context, c Comment) error {
	now := time.Now().UnixMilli()
	c.Utime = now
	c.Ctime = now
	return dao.db.
		WithContext(ctx).
		Create(&c).Error
}

type Comment struct {
	Id int64 `gorm:"column:id;primaryKey" json:"id"`
	// 发表评论的用户
	Uid int64 `gorm:"column:uid;index" json:"uid"`
	// 发表评论的业务类型
	Biz int32 `gorm:"column:biz;index:biz_type_id" json:"biz"`
	// 对应的业务ID
	BizID int64 `gorm:"column:biz_id;index:biz_type_id" json:"bizID"`
	// 根评论为0表示一级评论
	RootID sql.NullInt64 `gorm:"column:root_id;index" json:"rootID"`
	// 父级评论
	PID        sql.NullInt64 `gorm:"column:pid;index" json:"pid"`
	ReplyToUid int64         `json:"reply_to_uid"`
	// 外键 用于级联删除
	ParentComment *Comment `gorm:"ForeignKey:PID;AssociationForeignKey:ID;constraint:OnDelete:CASCADE"`
	// 评论内容
	Content string `gorm:"type:text;column:content" json:"content"`
	// 创建时间
	Ctime int64 `gorm:"column:ctime;" json:"ctime"`
	// 更新时间
	Utime int64 `gorm:"column:utime;" json:"utime"`
}
