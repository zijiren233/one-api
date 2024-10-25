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

func (s *Sealos) GetGroupRemainBalance(ctx context.Context, group string) (float64, PostGroupConsumer, error) {
	return 0, &SealosPostGroupConsumer{
		group: group,
		uid:   "",
	}, nil
}

type SealosPostGroupConsumer struct {
	group string
	uid   string
}

func (s *SealosPostGroupConsumer) PostGroupConsume(ctx context.Context, tokenName string, usage float64) error {
	return nil
}
