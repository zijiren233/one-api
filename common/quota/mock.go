package quota

var _ GroupQuota = (*MockGroupQuota)(nil)

var DefaultMockGroupQuota = NewMockGroupQuota()

type MockGroupQuota struct{}

func (q *MockGroupQuota) GetGroupQuota(id string) (int64, error) {
	return 10000000, nil
}

func NewMockGroupQuota() *MockGroupQuota {
	return &MockGroupQuota{}
}
