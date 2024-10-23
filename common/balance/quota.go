package balance

type GroupBalance interface {
	GetGroupRemainBalance(id string) (float64, error)
	PostGroupConsume(id string, amount float64) error
}

var Default = NewMockGroupBalance()
