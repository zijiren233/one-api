package balance

var _ GroupBalance = (*MockGroupBalance)(nil)

type MockGroupBalance struct{}

func NewMockGroupBalance() *MockGroupBalance {
	return &MockGroupBalance{}
}

func (q *MockGroupBalance) GetGroupRemainBalance(id string) (float64, error) {
	return 10000000, nil
}

func (q *MockGroupBalance) PostGroupConsume(id string, amount float64) error {
	return nil
}
