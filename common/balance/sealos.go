package balance

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	json "github.com/json-iterator/go"
	"github.com/shopspring/decimal"
)

var _ GroupBalance = (*Sealos)(nil)

var sealosHttpClient = http.Client{}

const (
	appType = "LLM-TOKEN"
)

type Sealos struct {
	accountUrl string
}

const (
	defaultAccountUrl = "http://account-service.account-system.svc.cluster.local:2333"
	balancePrecision  = 1000000
)

var decimalBalancePrecision = decimal.NewFromInt(balancePrecision)

func NewSealos(accountUrl string) *Sealos {
	if accountUrl == "" {
		accountUrl = defaultAccountUrl
	}
	return &Sealos{
		accountUrl: accountUrl,
	}
}

type sealosGetGroupBalanceReq struct {
	Workspace string `json:"workspace"`
}

type sealosGetGroupBalanceResp struct {
	Account struct {
		UserUid          string `json:"UserUID"`
		Balance          int64  `json:"Balance"`
		DeductionBalance int64  `json:"DeductionBalance"`
	} `json:"account"`
}

func newSealosGetGroupBalanceReq(group string) *sealosGetGroupBalanceReq {
	return &sealosGetGroupBalanceReq{
		Workspace: group,
	}
}

func (s *Sealos) GetGroupRemainBalance(ctx context.Context, group string) (float64, PostGroupConsumer, error) {
	sealosReq := newSealosGetGroupBalanceReq(group)
	reqBody, err := json.Marshal(sealosReq)
	if err != nil {
		return 0, nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/admin/v1alpha1/account-with-workspace", s.accountUrl), bytes.NewBuffer(reqBody))
	if err != nil {
		return 0, nil, err
	}
	resp, err := sealosHttpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	var sealosResp sealosGetGroupBalanceResp
	if err := json.NewDecoder(resp.Body).Decode(&sealosResp); err != nil {
		return 0, nil, err
	}
	return float64((sealosResp.Account.Balance - sealosResp.Account.DeductionBalance) / balancePrecision), &SealosPostGroupConsumer{
		accountUrl: s.accountUrl,
		group:      group,
		uid:        sealosResp.Account.UserUid,
	}, nil
}

type SealosPostGroupConsumer struct {
	accountUrl string
	group      string
	uid        string
}

type sealosPostGroupConsumeReq struct {
	Namespace string `json:"namespace"`
	Amount    int64  `json:"amount"`
	AppType   string `json:"appType"`
	AppName   string `json:"appName"`
	UserUID   string `json:"userUID"`
}

type sealosPostGroupConsumeResp struct {
	Message string `json:"message"`
}

func (s *SealosPostGroupConsumer) newSealosPostGroupConsumeReq(namespace string, amount int64, tokenName string) *sealosPostGroupConsumeReq {
	return &sealosPostGroupConsumeReq{
		Namespace: namespace,
		Amount:    amount,
		AppType:   appType,
		AppName:   tokenName,
		UserUID:   s.uid,
	}
}

var minConsumeAmount = decimal.NewFromInt(1)

func (s *SealosPostGroupConsumer) PostGroupConsume(ctx context.Context, tokenName string, usage float64) (float64, error) {
	u := decimal.NewFromFloat(usage).Mul(decimalBalancePrecision).Ceil()
	// Minimum consumption
	if u.LessThan(minConsumeAmount) {
		u = minConsumeAmount
	}
	sealosReq := s.newSealosPostGroupConsumeReq(s.group, u.IntPart(), tokenName)
	reqBody, err := json.Marshal(sealosReq)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/admin/v1alpha1/charge-billing", s.accountUrl), bytes.NewBuffer(reqBody))
	if err != nil {
		return 0, err
	}
	resp, err := sealosHttpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var sealosResp sealosPostGroupConsumeResp
	if err := json.NewDecoder(resp.Body).Decode(&sealosResp); err != nil {
		return 0, err
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("group (%s) consume failed with status code %d: %s", s.group, resp.StatusCode, sealosResp.Message)
	}
	return u.Div(decimalBalancePrecision).InexactFloat64(), nil
}
