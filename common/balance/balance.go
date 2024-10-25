package balance

import "context"

type GroupBalance interface {
	GetGroupRemainBalance(ctx context.Context, group string) (float64, error)
	PostGroupConsume(ctx context.Context, group string, tokenName string, usage float64) error
}

var Default GroupBalance = NewMockGroupBalance()
