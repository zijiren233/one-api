package quota

var _ GroupQuota = (*MockGroupQuota)(nil)

type MockGroupQuota struct{}

func NewMockGroupQuota() *MockGroupQuota {
	return &MockGroupQuota{}
}

func (q *MockGroupQuota) GetGroupRemainQuota(id string) (int64, error) {
	return 10000000, nil
}

func (q *MockGroupQuota) PostGroupConsume(id string, quota int64) error {
	return nil
}
