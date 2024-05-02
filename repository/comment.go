package repository

import (
	"context"
	"database/sql"
	"errors"
	commentv1 "github.com/MuxiKeStack/be-api/gen/proto/comment/v1"
	"github.com/MuxiKeStack/be-comment/domain"
	"github.com/MuxiKeStack/be-comment/pkg/logger"
	"github.com/MuxiKeStack/be-comment/repository/cache"
	"github.com/MuxiKeStack/be-comment/repository/dao"
	"github.com/ecodeclub/ekit/slice"
	"time"
)

var (
	ErrPermissionDenied = errors.New("没有该资源访问权限")
	ErrCommentNotFound  = dao.ErrRecordNotFound
)

type CommentRepository interface {
	FindByBiz(ctx context.Context, biz commentv1.Biz, bizId int64, curCommentId int64, limit int64) ([]domain.Comment, error)
	DeleteComment(ctx context.Context, commentId int64, uid int64) error
	GetCountByBiz(ctx context.Context, biz commentv1.Biz, bizId int64) (int64, error)
	GetMoreReplies(ctx context.Context, rid int64, curCommentId int64, limit int64) ([]domain.Comment, error)
	CreateCommentAsync(ctx context.Context, comment domain.Comment) error
	FindById(ctx context.Context, commentId int64) (domain.Comment, error)
	CreateCommentSync(ctx context.Context, comment domain.Comment) (int64, error)
}

type CachedCommentRepo struct {
	dao   dao.CommentDAO
	cache cache.CommentCache
	l     logger.Logger
}

func (repo *CachedCommentRepo) FindById(ctx context.Context, commentId int64) (domain.Comment, error) {
	comment, err := repo.dao.FindById(ctx, commentId)
	return repo.toDomain(comment), err
}

func NewCachedCommentRepo(dao dao.CommentDAO, cache cache.CommentCache, l logger.Logger) CommentRepository {
	return &CachedCommentRepo{
		dao:   dao,
		cache: cache,
		l:     l,
	}
}

func (repo *CachedCommentRepo) FindByBiz(ctx context.Context, biz commentv1.Biz, bizId int64, curCommentId int64, limit int64) ([]domain.Comment, error) {
	daoComments, err := repo.dao.FindByBiz(ctx, int32(biz), bizId, curCommentId, limit)
	return slice.Map(daoComments, func(idx int, src dao.Comment) domain.Comment {
		return repo.toDomain(src)
	}), err
}

func (repo *CachedCommentRepo) CreateCommentAsync(ctx context.Context, comment domain.Comment) error {
	// 这里可以做成异步
	_, err := repo.dao.Insert(ctx, repo.toEntity(comment))
	if err != nil {
		return err
	}
	return repo.cache.IncrBizCommentCountIfPresent(ctx, int32(comment.Biz), comment.BizId)
}

func (repo *CachedCommentRepo) CreateCommentSync(ctx context.Context, comment domain.Comment) (int64, error) {
	commentId, err := repo.dao.Insert(ctx, repo.toEntity(comment))
	if err != nil {
		return 0, err
	}
	err = repo.cache.IncrBizCommentCountIfPresent(ctx, int32(comment.Biz), comment.BizId)
	if err != nil {
		repo.l.Error("同步评论数缓存失败",
			logger.Error(err),
			logger.String("biz", comment.Biz.String()),
			logger.Int64("bizId", comment.BizId))

	}
	return commentId, nil
}

func (repo *CachedCommentRepo) DeleteComment(ctx context.Context, commentId int64, uid int64) error {
	comment, err := repo.dao.FindById(ctx, commentId)
	if comment.Uid != uid {
		return ErrPermissionDenied
	}
	// 要传入<biz,bizId>，因为delete也包括减少数目delete 'count'
	err = repo.dao.Delete(ctx, commentId, comment.Biz, comment.BizId)
	if err != nil {
		return err
	}
	return repo.cache.DecrBizCommentCountIfPresent(ctx, comment.Biz, comment.BizId)
}

func (repo *CachedCommentRepo) GetCountByBiz(ctx context.Context, biz commentv1.Biz, bizId int64) (int64, error) {
	count, err := repo.cache.GetBizCommentCount(ctx, int32(biz), bizId)
	if err == nil {
		return count, nil
	}
	if err != nil && err != cache.ErrKeyNotExists {
		repo.l.Error("获取评论数信息失败",
			logger.Error(err),
			logger.String("biz", biz.String()),
			logger.Int64("bizId", bizId),
		)
		// 降级，保护住数据库
		return 0, err
	}
	count, err = repo.dao.GetCountByBiz(ctx, int32(biz), bizId)
	if err != nil {
		return 0, err
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		er := repo.cache.SetBizCommentCount(ctx, int32(biz), bizId, count)
		if er != nil {
			if er != nil {
				repo.l.Error("回写评论数信息失败",
					logger.Error(err),
					logger.Any("biz", biz.String()),
					logger.Int64("bizId", bizId),
				)
			}
		}
	}()
	return count, nil
}

func (repo *CachedCommentRepo) GetMoreReplies(ctx context.Context, rid int64, curCommentId int64, limit int64) ([]domain.Comment, error) {
	cs, err := repo.dao.FindRepliesByRid(ctx, rid, curCommentId, limit)
	if err != nil {
		return nil, err
	}
	res := make([]domain.Comment, 0, len(cs))
	for _, cm := range cs {
		res = append(res, repo.toDomain(cm))
	}
	return res, nil
}

func (repo *CachedCommentRepo) toDomain(daoComment dao.Comment) domain.Comment {
	val := domain.Comment{
		Id: daoComment.Id,
		Commentator: domain.User{
			ID: daoComment.Uid,
		},
		Biz:        commentv1.Biz(daoComment.Biz),
		BizId:      daoComment.BizId,
		Content:    daoComment.Content,
		ReplyToUid: daoComment.ReplyToUid,
		CTime:      time.UnixMilli(daoComment.Ctime),
		UTime:      time.UnixMilli(daoComment.Utime),
	}
	if daoComment.PID.Valid {
		val.ParentComment = &domain.Comment{
			Id: daoComment.PID.Int64,
		}
	}
	if daoComment.RootID.Valid {
		val.RootComment = &domain.Comment{
			Id: daoComment.RootID.Int64,
		}
	}
	return val
}

func (repo *CachedCommentRepo) toEntity(domainComment domain.Comment) dao.Comment {
	daoComment := dao.Comment{
		Id:            domainComment.Id,
		Uid:           domainComment.Commentator.ID,
		Biz:           int32(domainComment.Biz),
		BizId:         domainComment.BizId,
		ReplyToUid:    domainComment.ReplyToUid,
		ParentComment: nil,
		Content:       domainComment.Content,
	}
	if domainComment.RootComment != nil {
		daoComment.RootID = sql.NullInt64{
			Valid: domainComment.RootComment.Id != 0,
			Int64: domainComment.RootComment.Id,
		}
	}
	if domainComment.ParentComment != nil {
		daoComment.PID = sql.NullInt64{
			Valid: domainComment.RootComment.Id != 0,
			Int64: domainComment.ParentComment.Id,
		}
	}
	return daoComment
}
