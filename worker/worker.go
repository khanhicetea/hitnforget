package worker

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/khanhicetea/hitnforget/database"
)

func Worker(wId int, queueName string, httpTimeOut time.Duration) {
	fmt.Println("Worker", wId, "is ready to work on queue", queueName, "...")

	rdb, ctx := database.NewRedis()
	client := &http.Client{
		Timeout: httpTimeOut,
	}

	for {
		popRequestID, err := rdb.LPop(ctx, queueName).Result()

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
			nextQueue := ""
			switch queueName {
			case "hnf:queue:normal":
				nextQueue = "hnf:queue:failed1"
			case "hnf:queue:failed1":
				nextQueue = "hnf:queue:failed2"
			}

			if nextQueue != "" {
				err = rdb.RPush(ctx, nextQueue, popRequestID).Err()
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
