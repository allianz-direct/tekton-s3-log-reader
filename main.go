package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type LogPayload struct {
	Date string `json:"date"`
	Log  string `json:"log"`
}

func GetSession(region string) *session.Session {
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	return sess
}

func GetObjects(bucket *string, prefix string, sess *session.Session) (*s3.ListObjectsV2Output, error) {
	svc := s3.New(sess)

	resp, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: bucket, Prefix: aws.String(prefix)})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func getObjectData(bucket *string, item string, sess *session.Session) string {
	svc := s3.New(sess)

	rawObject, err := svc.GetObject(
		&s3.GetObjectInput{
			Bucket: (bucket),
			Key:    aws.String(item),
		})

	if err != nil {
		log.Fatalf("Unable to get item %q, %v", item, err)
	}

	defer rawObject.Body.Close()
	body, err := ioutil.ReadAll(rawObject.Body)
	if err != nil {
		log.Print(err)
	}
	bodyString := fmt.Sprintf("%s", body)
	return bodyString
}

func main() {

	bucket := flag.String("b", os.Getenv("S3_BUCKET_NAME"), "Bucket name or S3_BUCKET_NAME env var")
	region := flag.String("r", os.Getenv("AWS_REGION"), "AWS Region or AWS_REGION env var")
	port := flag.String("p", ":5001", "Port")
	tls := flag.Bool("tls", false, "Use TLS to expose endpoint")
	serverCert := flag.String("cert", "", "TLS Certificate")
	serverKey := flag.String("key", "", "TLS Key")

	flag.Parse()

	sess := GetSession(*region)
	http.HandleFunc("/logs/", func(w http.ResponseWriter, r *http.Request) {
		prefix := strings.Split(r.URL.Path, "/logs/")
		var logPayload LogPayload
		objects, err := GetObjects(bucket, prefix[1], sess)
		if err != nil {
			log.Println("Got error retrieving list of objects:")
			log.Println(err)
			return
		}
		log.Print(prefix[1], " - ", *objects.KeyCount)
		for _, item := range objects.Contents {
			logUnformatted := strings.NewReader(getObjectData(bucket, *item.Key, sess))
			fscanner := bufio.NewScanner(logUnformatted)
			for fscanner.Scan() {
				err := json.Unmarshal(fscanner.Bytes(), &logPayload)
				if err != nil {
					fmt.Printf("%s", err)
					continue
				}
				fmt.Fprintf(w, logPayload.Log)
			}
		}
	})
	http.Handle("/metrics", promhttp.Handler())

	if *tls {
		if _, err := os.Stat(*serverCert); err != nil {
			fmt.Println("Error loading certificate:", err)
			panic(err)
		}
		if _, err := os.Stat(*serverKey); err != nil {
			fmt.Println("Error loading key:", err)
			panic(err)
		}
		log.Println("Listening TLS server on", *port)
		if err := http.ListenAndServeTLS(*port, *serverCert, *serverKey, nil); err != nil {
			fmt.Println("Failed to start the secure server:", err)
			panic(err)
		}
	} else {
		fmt.Println("Listening server on", *port)
		fmt.Println(http.ListenAndServe(*port, nil))
	}
}
