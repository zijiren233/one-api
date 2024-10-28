package balance

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	json "github.com/json-iterator/go"
	"github.com/shopspring/decimal"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/logger"
)

var _ GroupBalance = (*Sealos)(nil)

var sealosHttpClient = http.Client{}

const (
	defaultAccountUrl = "http://account-service.account-system.svc.cluster.local:2333"
	balancePrecision  = 1000000
	appType           = "LLM-TOKEN"
	sealosRequester   = "sealos-admin"
)

var (
	decimalBalancePrecision = decimal.NewFromInt(balancePrecision)
	jwtToken                string
)

func InitSealos(jwtKey string, accountUrl string) error {
	_jwtToken, err := newSealosToken(jwtKey)
	if err != nil {
		return fmt.Errorf("failed to generate sealos jwt token: %s", err)
	}
	jwtToken = _jwtToken
	Default = NewSealos(accountUrl)
	return nil
}

type Sealos struct {
	accountUrl string
}

func NewSealos(accountUrl string) *Sealos {
	if accountUrl == "" {
		accountUrl = defaultAccountUrl
	}
	return &Sealos{
		accountUrl: accountUrl,
	}
}

type sealosClaims struct {
	Requester string `json:"requester"`
	jwt.RegisteredClaims
}

func newSealosToken(key string) (string, error) {
	claims := &sealosClaims{
		Requester: sealosRequester,
		RegisteredClaims: jwt.RegisteredClaims{
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(common.StringToBytes(key))
}

type sealosGetGroupBalanceResp struct {
	Balance int64  `json:"balance"`
	UserUID string `json:"userUID"`
	Error   string `json:"error"`
}

func (s *Sealos) GetGroupRemainBalance(ctx context.Context, group string) (float64, PostGroupConsumer, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/admin/v1alpha1/account-with-workspace?namespace=%s", s.accountUrl, group), nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))
	resp, err := sealosHttpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	var sealosResp sealosGetGroupBalanceResp
	if err := json.NewDecoder(resp.Body).Decode(&sealosResp); err != nil {
		return 0, nil, err
	}
	if sealosResp.Error != "" {
		logger.Errorf(ctx, "get group (%s) balance failed: %s", group, sealosResp.Error)
		return 0, nil, fmt.Errorf("get group (%s) balance failed", group)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, nil, fmt.Errorf("get group (%s) balance failed with status code %d", group, resp.StatusCode)
	}
	return decimal.NewFromInt(sealosResp.Balance).Div(decimalBalancePrecision).InexactFloat64(), &SealosPostGroupConsumer{
		accountUrl: s.accountUrl,
		group:      group,
		uid:        sealosResp.UserUID,
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))
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
		logger.Errorf(ctx, "group (%s) consume failed with status code %d: %s", s.group, resp.StatusCode, sealosResp.Message)
		return 0, fmt.Errorf("group (%s) consume failed with status code %d", s.group, resp.StatusCode)
	}
	return u.Div(decimalBalancePrecision).InexactFloat64(), nil
}
