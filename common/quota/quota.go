package quota

type GroupQuota interface {
	GetGroupQuota(id string) (int64, error)
}
