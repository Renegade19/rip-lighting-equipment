package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Префикс ключей: lumicalc.jwt.blacklist.<токен>.
// Префикс с разделителем-точкой не даёт ключам разных сервисов совпасть
const jwtBlacklistPrefix = "lumicalc.jwt.blacklist."

// Client — чёрный список JWT в Redis. После logout токен попадает сюда
// и живёт ровно столько, сколько ему оставалось до истечения срока
type Client struct {
	client *redis.Client
}

func New(ctx context.Context, address, password string) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:        address,
		Password:    password,
		DialTimeout: 5 * time.Second,
		ReadTimeout: 3 * time.Second,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &Client{client: client}, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

// WriteJWTToBlacklist — logout: токен записывается в чёрный список.
// Значение ключа — логин проектировщика, чтобы в Redis было видно, чей это токен
func (c *Client) WriteJWTToBlacklist(ctx context.Context, jwtStr, login string, ttl time.Duration) error {
	return c.client.Set(ctx, jwtBlacklistPrefix+jwtStr, login, ttl).Err()
}

// CheckJWTInBlacklist — true, если проектировщик уже вышел с этим токеном
func (c *Client) CheckJWTInBlacklist(ctx context.Context, jwtStr string) (bool, error) {
	err := c.client.Get(ctx, jwtBlacklistPrefix+jwtStr).Err()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
