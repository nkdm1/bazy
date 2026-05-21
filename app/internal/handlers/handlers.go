package handlers

import "net/http"

func GetStatus(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok\n"))
}
