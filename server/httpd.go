package server

import (
	"fmt"
	"io"
	"net/http"

	"github.com/khanhicetea/hitnforget/database"
)

func queueHandle(w http.ResponseWriter, r *http.Request) {
	rdb, ctx := database.NewRedis()

	rID, err := database.QueueRequest(r, rdb, ctx, "hnf:queue:default")
	if err != nil {
		io.WriteString(w, "Error push to queue")
		return
	}

	io.WriteString(w, fmt.Sprintf("Queued request %d", rID))
}

func HTTPHandler() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/queue", queueHandle)

	return mux
}
