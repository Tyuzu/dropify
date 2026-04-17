package rdx

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var redis_url = os.Getenv("REDIS_URL")
var Conn = redis.NewClient(&redis.Options{
	Addr:     redis_url,
	Password: os.Getenv("REDIS_PASSWORD"), // no password set
	DB:       0,                           // use default DB
})

func InitRedis() { godotenv.Load() }

func RdxSet(ctx context.Context, key, value string) error {
	_, err := Conn.Set(ctx, key, value, 0).Result()
	if err != nil {
		return fmt.Errorf("error while doing SET command in redis : %v", err)
	}
	return nil
}

func RdxGet(ctx context.Context, key string) (string, error) {
	value, err := Conn.Get(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("error while doing GET command in redis : %v", err)
	}
	return value, nil
}

func RdxDel(ctx context.Context, key string) (int64, error) {
	value, err := Conn.Del(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("error while doing DEL command in redis : %v", err)
	}
	return value, nil
}

func RdxHset(ctx context.Context, hash, key, value string) error {
	_, err := Conn.HSet(ctx, hash, key, value).Result()
	if err != nil {
		return fmt.Errorf("error while doing HSET command in redis : %v", err)
	}
	return nil
}

func RdxHget(ctx context.Context, hash, key string) (string, error) {
	value, err := Conn.HGet(ctx, hash, key).Result()
	if err != nil {
		return "", err
	}
	return value, nil
}

func RdxHdel(ctx context.Context, hash, key string) (int64, error) {
	value, err := Conn.HDel(ctx, hash, key).Result()
	if err != nil {
		return 0, fmt.Errorf("error while doing HDEL command in redis : %v", err)
	}
	return value, nil
}

func RdxHgetall(ctx context.Context, hash string) (map[string]string, error) {
	value, err := Conn.HGetAll(ctx, hash).Result()
	if err != nil {
		return nil, fmt.Errorf("error while doing HGETALL command in redis : %v", err)
	}
	return value, nil
}

func RdxAppend(ctx context.Context, key, value string) error {
	_, err := Conn.Append(ctx, key, value).Result()
	if err != nil {
		return fmt.Errorf("error while doing APPEND command in redis : %v", err)
	}
	return nil
}

func SetWithExpiry(ctx context.Context, key, value string, exptime time.Duration) error {
	_, err := Conn.Set(ctx, key, value, exptime).Result()
	if err != nil {
		return fmt.Errorf("error while SetWithExpiry in redis : %v", err)
	}
	return nil
}

func Exists(ctx context.Context, key string) (bool, error) {
	exists, err := Conn.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("error while checking EXISTS in redis : %v", err)
	}
	return exists > 0, nil
}

func RdxSetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	success, err := Conn.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("error while doing SETNX command in redis : %v", err)
	}
	return success, nil
}
