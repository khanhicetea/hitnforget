package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	cli "github.com/urfave/cli/v2"
)

func matchPattern(pattern string, s string) bool {
	if pattern == "*" {
		return true
	}
	matched, _ := filepath.Match(pattern, s)
	return matched
}

func newRedis(ctx *cli.Context) (*redis.Client, context.Context) {
	redisAddr := ctx.Value("redis").(string)

	return redis.NewClient(&redis.Options{
		Addr: redisAddr,
	}), context.Background()
}

func queueRedisKey(queueName string) string {
	return fmt.Sprintf("hnf:queue:%s", queueName)
}

func queueRedisChannel(queueName string) string {
	return fmt.Sprintf("hnf:channel:%s", queueName)
}

func queueRequest(r *http.Request, rdb *redis.Client, ctx context.Context, queueName string) (int64, error) {
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

func getRequestFromQueue(rID string, rdb *redis.Client, ctx context.Context) (*http.Request, error) {
	rawRequest, _ := rdb.Get(ctx, "hnf:request:"+rID).Result()
	reader := strings.NewReader(rawRequest)
	buf := bufio.NewReader(reader)
	req, _ := http.ReadRequest(buf)

	return req, nil
}

func httpHandler(c *cli.Context) *http.ServeMux {
	mux := http.NewServeMux()

	rdb, ctx := newRedis(c)
	defaultChannel := queueRedisChannel("default")
	pubsub := rdb.Subscribe(ctx, defaultChannel)
	defer pubsub.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	mux.HandleFunc("/_queue", func(w http.ResponseWriter, r *http.Request) {
		rID, err := queueRequest(r, rdb, ctx, "hnf:queue:default")
		if err != nil {
			io.WriteString(w, "Error push to queue")
			return
		}
		rdb.Publish(ctx, defaultChannel, rID)
		io.WriteString(w, fmt.Sprintf("Queued request %d", rID))
	})

	return mux
}

func runWorker(c *cli.Context, wId int, workingQueue string, failedQueue string) {
	fmt.Println("Worker", wId, "is ready to work on queue", workingQueue, "...")

	rdb, ctx := newRedis(c)
	tr := &http.Transport{
		MaxIdleConnsPerHost: 30,
		TLSHandshakeTimeout: time.Minute,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   time.Minute,
	}

	pubsub := rdb.Subscribe(ctx, queueRedisChannel(workingQueue))
	defer pubsub.Close()

	redisKeyWorkingQueue := queueRedisKey(workingQueue)
	redisKeyFailedQueue := queueRedisKey(failedQueue)

	for {
		popRequestID, err := rdb.LPop(ctx, redisKeyWorkingQueue).Result()
		if err != nil {
			_, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				time.Sleep(time.Second)
			}
			continue
		}

		fmt.Println("Worker", wId, "is working on request", popRequestID)

		origReq, _ := getRequestFromQueue(popRequestID, rdb, ctx)
		origUrl := origReq.Header.Get("X-Hnf-Url")
		origMethod := origReq.Header.Get("X-Hnf-Method")
		expectedStatusCode := origReq.Header.Get("X-Hnf-Expected-Status")

		if origUrl == "" {
			rdb.Del(ctx, "hnf:request:"+popRequestID)
			fmt.Printf("Request %s has no URL, skipping\n", popRequestID)
			continue
		}

		if origMethod == "" {
			origMethod = "GET"
		}
		if expectedStatusCode == "" {
			expectedStatusCode = "2**"
		}

		doReq, _ := http.NewRequest(origMethod, origUrl, origReq.Body)

		for key, values := range origReq.Header {
			if !strings.Contains(key, "X-Hnf-") {
				for _, value := range values {
					doReq.Header.Add(key, value)
				}
			}
		}

		doReq.Header.Set("Connection", "Close")

		resp, err := client.Do(doReq)
		if err != nil || !matchPattern(expectedStatusCode, fmt.Sprint(resp.StatusCode)) {
			if failedQueue != "" {
				fmt.Println("Delegate to", failedQueue, "the request", popRequestID)
				err = rdb.RPush(ctx, redisKeyFailedQueue, popRequestID).Err()
				if err != nil {
					panic(err)
				}
				rdb.Publish(ctx, queueRedisChannel(failedQueue), popRequestID)
				continue
			} else {
				fmt.Println("Drop the request", popRequestID)
			}
		}

		rdb.Del(ctx, "hnf:request:"+popRequestID)
		fmt.Printf("Request %s has status code = %d , deleted #%s requests from queue\n", popRequestID, resp.StatusCode, popRequestID)
	}

}

func main() {
	app := &cli.App{
		Name:        "HitNForget",
		Description: "HTTP Later Server",
		Commands: []*cli.Command{
			{
				Name:    "server",
				Aliases: []string{"s"},
				Usage:   "Run HTTP queue server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "bind",
						Usage: "Binding address",
						Value: "127.0.0.1",
					},
					&cli.StringFlag{
						Name:  "port",
						Usage: "Binding port",
						Value: "8080",
					},
					&cli.StringFlag{
						Name:  "redis",
						Usage: "Redis Addr",
						Value: "127.0.0.1:6379",
					},
				},
				Action: func(c *cli.Context) error {
					bind := c.Value("bind").(string)
					port := c.Value("port").(string)
					fmt.Printf("Running QUEUE server on %s:%s ...", bind, port)
					http.ListenAndServe(net.JoinHostPort(bind, port), httpHandler(c))
					return nil
				},
			},
			{
				Name:    "worker",
				Aliases: []string{"w"},
				Usage:   "Run worker",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "working_queue",
						Usage: "Working queue name",
						Value: "default",
					},
					&cli.StringFlag{
						Name:  "failed_queue",
						Usage: "Next failed queue",
						Value: "",
					},
					&cli.StringFlag{
						Name:  "redis",
						Usage: "Redis Addr",
						Value: "127.0.0.1:6379",
					},
				},
				Action: func(c *cli.Context) error {
					workingQueue := c.Value("working_queue").(string)
					failedQueue := c.Value("failed_queue").(string)
					s1 := rand.NewSource(time.Now().UnixNano())
					r1 := rand.New(s1)
					workerId := r1.Intn(10000)
					runWorker(c, workerId, workingQueue, failedQueue)
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
