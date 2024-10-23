package balance

import "context"

var _ GroupBalance = (*MockGroupBalance)(nil)

type MockGroupBalance struct{}

func NewMockGroupBalance() *MockGroupBalance {
	return &MockGroupBalance{}
}

func (q *MockGroupBalance) GetGroupRemainBalance(ctx context.Context, group string) (float64, error) {
	return 10000000, nil
}

func (q *MockGroupBalance) PostGroupConsume(ctx context.Context, group string, usage float64) error {
	return nil
}
