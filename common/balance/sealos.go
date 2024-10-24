package balance

import "context"

var _ GroupBalance = (*Sealos)(nil)

type Sealos struct {
	accountUrl string
}

func NewSealos(accountUrl string) *Sealos {
	return &Sealos{
		accountUrl: accountUrl,
	}
}

func (s *Sealos) GetGroupRemainBalance(ctx context.Context, group string) (float64, error) {
	return 0, nil
}

func (s *Sealos) PostGroupConsume(ctx context.Context, group string, usage float64) error {
	return nil
}
