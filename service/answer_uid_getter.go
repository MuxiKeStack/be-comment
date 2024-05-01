package service

import (
	"context"
	answerv1 "github.com/MuxiKeStack/be-api/gen/proto/answer/v1"
)

type AnswerUIDGetter struct {
	answerClient answerv1.AnswerServiceClient
}

func (a *AnswerUIDGetter) GetUID(ctx context.Context, bizId int64) (int64, error) {
	res, err := a.answerClient.Detail(ctx, &answerv1.DetailRequest{AnswerId: bizId})
	if err != nil {
		return 0, err
	}
	return res.GetAnswer().GetPublisherId(), nil
}
