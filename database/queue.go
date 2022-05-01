package database

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/go-redis/redis/v8"
)

func QueueRequest(r *http.Request, rdb *redis.Client, ctx context.Context, queueName string) (int64, error) {
	dr, _ := httputil.DumpRequest(r, true)

	rID, err := rdb.Incr(ctx, "hnf:requests_num").Result()
	if err != nil {
		return 0, err
	}

	err = rdb.Set(ctx, "hnf:request:"+fmt.Sprint(rID), dr, 0).Err()
	if err != nil {
		return 0, err
	}

	err = rdb.RPush(ctx, queueName, rID).Err()
	if err != nil {
		return 0, err
	}

	return rID, nil
}

func GetRequestFromQueue(rID string, rdb *redis.Client, ctx context.Context) (*http.Request, error) {
	rawRequest, _ := rdb.Get(ctx, "hnf:request:"+rID).Result()
	reader := strings.NewReader(rawRequest)
	buf := bufio.NewReader(reader)
	req, _ := http.ReadRequest(buf)

	return req, nil
}
