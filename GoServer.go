package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/jamespearly/loggly"
)

type EndPointRequest struct {
	SystemTime string
	Status     int
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewStatusResponseWriter(w http.ResponseWriter) *statusResponseWriter {
	return &statusResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (sw *statusResponseWriter) WriteHeader(statusCode int) {
	sw.statusCode = statusCode
	sw.ResponseWriter.WriteHeader(statusCode)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	req := EndPointRequest{
		SystemTime: time.Now().Format(time.RFC3339),
		Status:     http.StatusOK,
	}

	reqJSON, _ := json.Marshal(req)
	w.Write([]byte(reqJSON))
}

func catchAllHandler(w http.ResponseWriter, r *http.Request) {
	// Do Nothing...
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := NewStatusResponseWriter(w)

		if r.Method != "GET" {
			sw.WriteHeader(http.StatusMethodNotAllowed)
		} else if r.RequestURI != "/jwilcox5/status" {
			sw.WriteHeader(http.StatusNotFound)
		}

		logTag := "IQAir Air Quality Data"

		client := loggly.New(logTag)

		logErr := client.EchoSend("info", "\nMethod Type: "+r.Method+"\nSource IP Address: "+r.RequestURI+"\nRequest Path: "+r.Host+"\nHTTP Status Code: "+strconv.Itoa(sw.statusCode))
		fmt.Println("err:", logErr)

		next.ServeHTTP(w, r)
	})
}

func main() {
	r := mux.NewRouter()
	r.Use(loggingMiddleware)
	r.HandleFunc("/", catchAllHandler)
	r.HandleFunc("/{path}", catchAllHandler)
	r.HandleFunc("/jwilcox5/status", statusHandler).Methods("GET")
	r.HandleFunc("/jwilcox5/{path}", catchAllHandler)
	http.Handle("/", r)
	http.ListenAndServe(":35000", r)
}
