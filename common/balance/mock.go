package balance

import "context"

var _ GroupBalance = (*MockGroupBalance)(nil)

type MockGroupBalance struct{}

func NewMockGroupBalance() *MockGroupBalance {
	return &MockGroupBalance{}
}

func (q *MockGroupBalance) GetGroupRemainBalance(ctx context.Context, group string) (float64, PostGroupConsumer, error) {
	return 10000000, q, nil
}

func (q *MockGroupBalance) PostGroupConsume(ctx context.Context, tokenName string, usage float64) (float64, error) {
	return usage, nil
}
