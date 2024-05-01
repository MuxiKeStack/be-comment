package service

import (
	"context"
	evaluationv1 "github.com/MuxiKeStack/be-api/gen/proto/evaluation/v1"
)

type EvaluationUIDGetter struct {
	evaluationClient evaluationv1.EvaluationServiceClient
}

func (e *EvaluationUIDGetter) GetUID(ctx context.Context, bizId int64) (int64, error) {
	res, err := e.evaluationClient.Detail(ctx, &evaluationv1.DetailRequest{EvaluationId: bizId})
	if err != nil {
		return 0, err
	}
	return res.GetEvaluation().GetPublisherId(), nil
}
