package main

import (
	"net/http"

	"log"

	"encoding/json"

	"github.com/couchbase/gocb"
	"github.com/gorilla/mux"
	hashids "github.com/speps/go-hashids"

	"time"
)

type MyUrl struct {
	ID       string `json:"id,omitempty"`
	LongUrl  string `json:"longUrl,omitempty"`
	ShortUrl string `json:"shortUrl,omitempty"`
}

var bucket *gocb.Bucket
var bucketName string

func ExpandEndpoint(w http.ResponseWriter, req *http.Request) {
	var n1qlParams []interface{}
	query := gocb.NewN1qlQuery("SELECT `" + bucketName + "`.* FROM `" + bucketName + "` WHERE shortUrl = $1")
	params := req.URL.Query()
	n1qlParams = append(n1qlParams, params.Get("shortUrl"))
	rows, _ := bucket.ExecuteN1qlQuery(query, params)
	var row MyUrl
	rows.One(&row)
	json.NewEncoder(w).Encode(row)
}

func CreateEndpoint(w http.ResponseWriter, req *http.Request) {
	var url MyUrl
	_ = json.NewDecoder(req.Body).Decode(&url)
	var n1qlParams []interface{}
	n1qlParams = append(n1qlParams, url.LongUrl)
	query := gocb.NewN1qlQuery("SELECT `" + bucketName + "`.* FROM `" + bucketName + "` WHERE longUrl = $1")
	rows, err := bucket.ExecuteN1qlQuery(query, n1qlParams)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte(err.Error()))
		return
	}
	var row MyUrl
	rows.One(&row)
	if row == (MyUrl{}) {
		hd := hashids.NewData()
		h := hashids.NewWithData(hd)
		now := time.Now()
		url.ID, _ = h.Encode([]int{int(now.Unix())})
		url.ShortUrl = "http://localhost:8000/" + url.ID
		bucket.Insert(url.ID, url, 0)
	} else {
		url = row
	}
	json.NewEncoder(w).Encode(url)
}

func RootEndpoint(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	var url MyUrl
	bucket.Get(params["id"], &url)
	http.Redirect(w, req, url.LongUrl, 301)
}

func main() {
	router := mux.NewRouter()
	cluster, _ := gocb.Connect("couchbase://127.0.0.1")
	bucketName = "example"
	bucket, _ = cluster.OpenBucket(bucketName, "")
	router.HandleFunc("/create", CreateEndpoint).Methods("PUT")
	router.HandleFunc("/expand/", ExpandEndpoint).Methods("GET")
	router.HandleFunc("/{id}", RootEndpoint).Methods("GET")
	log.Fatal(http.ListenAndServe(":8000", router))
}
