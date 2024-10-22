package quota

type GroupQuota interface {
	GetGroupRemainQuota(id string) (int64, error)
	PostGroupConsume(id string, quota int64) error
}

var Default = NewMockGroupQuota()
