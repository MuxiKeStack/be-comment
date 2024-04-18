package service

import (
	"context"
	commentv1 "github.com/MuxiKeStack/be-api/gen/proto/comment/v1"
	"github.com/MuxiKeStack/be-comment/domain"
	"github.com/MuxiKeStack/be-comment/repository"
)

type CommentService interface {
	CreateComment(ctx context.Context, comment domain.Comment) error
	GetCommentList(ctx context.Context, biz commentv1.Biz, bizId int64, curCommentId int64, limit int64) ([]domain.Comment, error)
	DeleteComment(ctx context.Context, commentId int64, uid int64) error
	GetMoreReplies(ctx context.Context, rid int64, curCommentId int64, limit int64) ([]domain.Comment, error)
	Count(ctx context.Context, biz commentv1.Biz, bizId int64) (int64, error)
}

type commentService struct {
	repo repository.CommentRepository
}

func NewCommentService(repo repository.CommentRepository) CommentService {
	return &commentService{repo: repo}
}

func (s *commentService) GetCommentList(ctx context.Context, biz commentv1.Biz, bizId int64, curCommentId int64, limit int64) ([]domain.Comment, error) {
	list, err := s.repo.FindByBiz(ctx, biz, bizId, curCommentId, limit)
	if err != nil {
		return nil, err
	}
	return list, err
}

func (s *commentService) DeleteComment(ctx context.Context, commentId int64, uid int64) error {
	return s.repo.DeleteComment(ctx, commentId, uid)
}

func (s *commentService) Count(ctx context.Context, biz commentv1.Biz, bizId int64) (int64, error) {
	return s.repo.GetCountByBiz(ctx, biz, bizId)
}

func (c *commentService) GetMoreReplies(ctx context.Context, rid int64, curCommentId int64, limit int64) ([]domain.Comment, error) {
	return c.repo.GetMoreReplies(ctx, rid, curCommentId, limit)
}

func (c *commentService) CreateComment(ctx context.Context, comment domain.Comment) error {
	// 要去聚合一下 replyToUid
	if comment.ParentComment.Id != 0 {
		// 有父评论，找到父评论的发布者
		pc, err := c.repo.FindById(ctx, comment.ParentComment.Id)
		if err != nil {
			return err
		}
		comment.ReplyToUid = pc.Commentator.ID
	}
	return c.repo.CreateComment(ctx, comment)
}

func NewCommentSvc(repo repository.CommentRepository) CommentService {
	return &commentService{
		repo: repo,
	}
}
