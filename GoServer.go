package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"

	"github.com/jamespearly/loggly"
)

type AirQualityData struct {
	DateTime string `json:"datetime"`
	Status   string `json:"status"`
	City     string `json:"city"`
	State    string `json:"state"`
	Country  string `json:"country"`
	Data     struct {
		City     string `json:"city"`
		State    string `json:"state"`
		Country  string `json:"country"`
		Location struct {
			Type        string    `json:"type"`
			Coordinates []float64 `json:"coordinates"`
		} `json:"location"`
		Current struct {
			Pollution struct {
				Ts     time.Time `json:"ts"`
				Aqius  int       `json:"aqius"`
				Mainus string    `json:"mainus"`
				Aqicn  int       `json:"aqicn"`
				Maincn string    `json:"maincn"`
			} `json:"pollution"`
			Weather struct {
				Ts time.Time `json:"ts"`
				Tp int       `json:"tp"`
				Pr int       `json:"pr"`
				Hu int       `json:"hu"`
				Ws float64   `json:"ws"`
				Wd int       `json:"wd"`
				Ic string    `json:"ic"`
			} `json:"weather"`
		} `json:"current"`
	} `json:"data"`
}

type StatusRequest struct {
	Table       string
	RecordCount int
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

func InitializeSess() *dynamodb.DynamoDB {
	awsSess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := dynamodb.New(awsSess)

	return svc
}

func GetTableItemsAndName(svc *dynamodb.DynamoDB) (*dynamodb.ScanOutput, string) {
	tablesInput := &dynamodb.ListTablesInput{}

	tablesOutput, tableErr := svc.ListTables(tablesInput)

	if tableErr != nil {
		panic(tableErr)
	}

	var myTableName string

	for _, n := range tablesOutput.TableNames {
		if *n == "air-quality-data-jwilcox5" {
			myTableName = *n
		}
	}

	items, itemErr := svc.Scan(&dynamodb.ScanInput{
		TableName: aws.String(myTableName),
	})

	if itemErr != nil {
		panic(itemErr)
	}

	return items, myTableName
}

func allHandler(w http.ResponseWriter, r *http.Request) {
	svc := InitializeSess()

	items, _ := GetTableItemsAndName(svc)

	for _, i := range items.Items {
		item := AirQualityData{}

		itemErr := dynamodbattribute.UnmarshalMap(i, &item)

		if itemErr != nil {
			log.Fatalf("Got error unmarshalling: %s", itemErr)
		}

		itemJSON, _ := json.Marshal(item)
		w.Write([]byte(itemJSON))
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	svc := InitializeSess()

	tableResponse, tableErr := svc.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String("air-quality-data-jwilcox5"),
	})

	if tableErr != nil {
		panic(tableErr)
	}

	recCount := int(*tableResponse.Table.ItemCount)

	statusReq := StatusRequest{
		Table:       "air-quality-data-jwilcox5",
		RecordCount: recCount,
	}

	statusJSON, _ := json.Marshal(statusReq)
	w.Write([]byte(statusJSON))
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	re := regexp.MustCompile("^[-a-zA-Z0-9:]+$")

	query := r.URL.Query()

	param1, paramPresent1 := query["city"]

	if !paramPresent1 || len(param1[0]) == 0 || len(param1[0]) > 20 {
		w.Write([]byte("400 - The query parameter either does not exist, is empty, or is too long"))
	} else if len(query) > 1 {
		w.Write([]byte("400 - There are too many query parameters"))
	} else if !(re.MatchString(param1[0])) {
		w.Write([]byte("400 - The query parameter contains characters that should not be in the parameter"))
	} else {
		svc := InitializeSess()

		_, myTableName := GetTableItemsAndName(svc)

		targetCity := param1[0]

		filter := expression.Name("city").Equal(expression.Value(targetCity))

		expr, exprErr := expression.NewBuilder().WithFilter(filter).Build()

		if exprErr != nil {
			log.Fatalf("Got error building expression: %s", exprErr)
		}

		params := &dynamodb.ScanInput{
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			FilterExpression:          expr.Filter(),
			TableName:                 aws.String(myTableName),
		}

		queryResults, paramErr := svc.Scan(params)

		if paramErr != nil {
			log.Fatalf("Query API call failed: %s", paramErr)
		}

		for _, i := range queryResults.Items {
			item := AirQualityData{}

			itemErr := dynamodbattribute.UnmarshalMap(i, &item)

			if itemErr != nil {
				log.Fatalf("Got error unmarshalling: %s", itemErr)
			}

			itemJSON, _ := json.Marshal(item)
			w.Write([]byte(itemJSON))
		}
	}
}

func catchAllHandler(w http.ResponseWriter, r *http.Request) {
	// Do Nothing...
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := NewStatusResponseWriter(w)

		if r.Method != "GET" {
			sw.WriteHeader(http.StatusMethodNotAllowed)
		} else if !(strings.Contains(r.RequestURI, "all")) && !(strings.Contains(r.RequestURI, "status")) && !(strings.Contains(r.RequestURI, "search")) {
			sw.WriteHeader(http.StatusNotFound)
		} else if strings.Contains(r.RequestURI, "search") {
			re := regexp.MustCompile("^[a-zA-Z:]+$")

			query := r.URL.Query()

			param1, paramPresent1 := query["city"]

			if !paramPresent1 || len(param1[0]) == 0 || len(param1[0]) > 20 {
				sw.WriteHeader(http.StatusBadRequest)
			} else if len(query) > 1 {
				sw.WriteHeader(http.StatusBadRequest)
			} else if !(re.MatchString(param1[0])) {
				sw.WriteHeader(http.StatusBadRequest)
			}
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
	r.HandleFunc("/jwilcox5/all", allHandler).Methods("GET")
	r.HandleFunc("/jwilcox5/status", statusHandler).Methods("GET")
	r.HandleFunc("/jwilcox5/search", searchHandler).Methods("GET")
	r.HandleFunc("/jwilcox5/{path}", catchAllHandler)
	r.HandleFunc("/{path}/{path}", catchAllHandler)
	http.Handle("/", r)
	http.ListenAndServe(":8080", r)
}
