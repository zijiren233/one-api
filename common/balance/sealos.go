package balance

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	json "github.com/json-iterator/go"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/logger"
)

const (
	defaultAccountUrl     = "http://account-service.account-system.svc.cluster.local:2333"
	balancePrecision      = 1000000
	appType               = "LLM-TOKEN"
	sealosRequester       = "sealos-admin"
	sealosGroupBalanceKey = "sealos:balance:%s"
)

var (
	_                       GroupBalance = (*Sealos)(nil)
	sealosHttpClient                     = &http.Client{}
	decimalBalancePrecision              = decimal.NewFromInt(balancePrecision)
	minConsumeAmount                     = decimal.NewFromInt(1)
	jwtToken                string
)

type Sealos struct {
	accountUrl string
}

func InitSealos(jwtKey string, accountUrl string) error {
	token, err := newSealosToken(jwtKey)
	if err != nil {
		return fmt.Errorf("failed to generate sealos jwt token: %s", err)
	}
	jwtToken = token
	Default = NewSealos(accountUrl)
	return nil
}

func NewSealos(accountUrl string) *Sealos {
	if accountUrl == "" {
		accountUrl = defaultAccountUrl
	}
	return &Sealos{accountUrl: accountUrl}
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

type sealosCache struct {
	Balance int64  `redis:"b"`
	UserUID string `redis:"u"`
}

func cacheSetGroupBalance(ctx context.Context, group string, balance int64, userUID string) error {
	if !common.RedisEnabled {
		return nil
	}
	pipe := common.RDB.Pipeline()
	pipe.HSet(ctx, fmt.Sprintf(sealosGroupBalanceKey, group), sealosCache{
		Balance: balance,
		UserUID: userUID,
	})
	pipe.Expire(ctx, fmt.Sprintf(sealosGroupBalanceKey, group), time.Second*3)
	_, err := pipe.Exec(ctx)
	return err
}

func cacheGetGroupBalance(ctx context.Context, group string) (*sealosCache, error) {
	if !common.RedisEnabled {
		return nil, redis.Nil
	}
	var cache sealosCache
	if err := common.RDB.HGetAll(ctx, fmt.Sprintf(sealosGroupBalanceKey, group)).Scan(&cache); err != nil {
		return nil, err
	}
	return &cache, nil
}

var decreaseGroupBalanceScript = redis.NewScript(`
	local balance = redis.call("HGet", KEYS[1], "balance")
	if balance == false then
		return redis.status_reply("ok")
	end
	redis.call("HSet", KEYS[1], "balance", balance - ARGV[1])
	return redis.status_reply("ok")
`)

func cacheDecreaseGroupBalance(ctx context.Context, group string, amount int64) error {
	if !common.RedisEnabled {
		return nil
	}
	return decreaseGroupBalanceScript.Run(ctx, common.RDB, []string{fmt.Sprintf(sealosGroupBalanceKey, group)}, amount).Err()
}

// GroupBalance interface implementation
func (s *Sealos) GetGroupRemainBalance(ctx context.Context, group string) (float64, PostGroupConsumer, error) {
	if cache, err := cacheGetGroupBalance(ctx, group); err == nil && cache.UserUID != "" {
		return decimal.NewFromInt(cache.Balance).Div(decimalBalancePrecision).InexactFloat64(),
			newSealosPostGroupConsumer(s.accountUrl, group, cache.UserUID), nil
	} else if err != nil && err != redis.Nil {
		logger.Errorf(ctx, "get group (%s) balance cache failed: %s", group, err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	balance, userUID, err := s.fetchBalanceFromAPI(ctx, group)
	if err != nil {
		return 0, nil, err
	}

	if err := cacheSetGroupBalance(ctx, group, balance, userUID); err != nil {
		logger.Errorf(ctx, "set group (%s) balance cache failed: %s", group, err)
	}

	return decimal.NewFromInt(balance).Div(decimalBalancePrecision).InexactFloat64(),
		newSealosPostGroupConsumer(s.accountUrl, group, userUID), nil
}

func (s *Sealos) fetchBalanceFromAPI(ctx context.Context, group string) (balance int64, userUID string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/admin/v1alpha1/account-with-workspace?namespace=%s", s.accountUrl, group), nil)
	if err != nil {
		return 0, "", err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))
	resp, err := sealosHttpClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	var sealosResp sealosGetGroupBalanceResp
	if err := json.NewDecoder(resp.Body).Decode(&sealosResp); err != nil {
		return 0, "", err
	}

	if sealosResp.Error != "" {
		logger.Errorf(ctx, "get group (%s) balance failed: %s", group, sealosResp.Error)
		return 0, "", fmt.Errorf("get group (%s) balance failed", group)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("get group (%s) balance failed with status code %d", group, resp.StatusCode)
	}

	return sealosResp.Balance, sealosResp.UserUID, nil
}

type SealosPostGroupConsumer struct {
	accountUrl string
	group      string
	uid        string
}

func newSealosPostGroupConsumer(accountUrl, group, uid string) *SealosPostGroupConsumer {
	return &SealosPostGroupConsumer{
		accountUrl: accountUrl,
		group:      group,
		uid:        uid,
	}
}

func (s *SealosPostGroupConsumer) PostGroupConsume(ctx context.Context, tokenName string, usage float64) (float64, error) {
	amount := s.calculateAmount(usage)

	if err := s.postConsume(ctx, amount.IntPart(), tokenName); err != nil {
		return 0, err
	}

	if err := cacheDecreaseGroupBalance(ctx, s.group, amount.IntPart()); err != nil {
		logger.Errorf(ctx, "decrease group (%s) balance cache failed: %s", s.group, err)
	}

	return amount.Div(decimalBalancePrecision).InexactFloat64(), nil
}

func (s *SealosPostGroupConsumer) calculateAmount(usage float64) decimal.Decimal {
	amount := decimal.NewFromFloat(usage).Mul(decimalBalancePrecision).Ceil()
	if amount.LessThan(minConsumeAmount) {
		amount = minConsumeAmount
	}
	return amount
}

func (s *SealosPostGroupConsumer) postConsume(ctx context.Context, amount int64, tokenName string) error {
	reqBody, err := json.Marshal(sealosPostGroupConsumeReq{
		Namespace: s.group,
		Amount:    amount,
		AppType:   appType,
		AppName:   tokenName,
		UserUID:   s.uid,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/admin/v1alpha1/charge-billing", s.accountUrl), bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))
	resp, err := sealosHttpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var sealosResp sealosPostGroupConsumeResp
	if err := json.NewDecoder(resp.Body).Decode(&sealosResp); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		logger.Errorf(ctx, "group (%s) consume failed with status code %d: %s",
			s.group, resp.StatusCode, sealosResp.Message)
		return fmt.Errorf("group (%s) consume failed with status code %d", s.group, resp.StatusCode)
	}

	return nil
}
