package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/khanhicetea/hitnforget/server"
	"github.com/khanhicetea/hitnforget/worker"
)

func main() {
	fmt.Println("Hit N Forget")

	server := server.NewServer()

	go worker.Worker(1, "hnf:queue:normal", time.Minute*1)
	go worker.Worker(2, "hnf:queue:normal", time.Minute*1)
	go worker.Worker(3, "hnf:queue:normal", time.Minute*1)
	go worker.Worker(11, "hnf:queue:failed1", time.Minute*2)
	go worker.Worker(21, "hnf:queue:failed2", time.Minute*3)

	err := http.ListenAndServe("127.0.0.1:3333", server)
	if err != nil {
		log.Fatal(err)
	}
}
