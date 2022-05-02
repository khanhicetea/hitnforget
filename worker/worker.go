package worker

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/khanhicetea/hitnforget/database"
)

func Worker(wId int, workingQueue string, failedQueue string) {
	fmt.Println("Worker", wId, "is ready to work on queue", workingQueue, "...")

	rdb, ctx := database.NewRedis()
	client := &http.Client{
		Timeout: time.Minute,
	}

	redisKeyWorkingQueue := database.QueueRedisKey(workingQueue)
	redisKeyFailedQueue := database.QueueRedisKey(failedQueue)

	for {
		popRequestID, err := rdb.LPop(ctx, redisKeyWorkingQueue).Result()

		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		fmt.Println("Worker", wId, "is working on request", popRequestID)

		origReq, _ := database.GetRequestFromQueue(popRequestID, rdb, ctx)

		origUrl := origReq.Header.Get("X-Hnf-Url")
		origMethod := origReq.Header.Get("X-Hnf-Method")
		doReq, _ := http.NewRequest(origMethod, origUrl, origReq.Body)

		for key, values := range origReq.Header {
			if !strings.Contains(key, "X-Hnf-") {
				for _, value := range values {
					doReq.Header.Add(key, value)
				}
			}
		}

		resp, err := client.Do(doReq)
		if err != nil || resp.StatusCode >= 400 {
			if failedQueue != "" {
				fmt.Println("Delegate to", failedQueue, "the request", popRequestID)
				err = rdb.RPush(ctx, redisKeyFailedQueue, popRequestID).Err()
				if err != nil {
					panic(err)
				}
				continue
			}

		}

		rdb.Del(ctx, "hnf:request:"+popRequestID)
		fmt.Printf("Request %s has status code = %d , deleted #%s requests from queue\n", popRequestID, resp.StatusCode, popRequestID)
	}

}
