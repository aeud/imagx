package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/nfnt/resize"
	"html"
	"image"
	_ "image/gif"
	jpeg "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const (
	PORT int = 8080
)

func catchError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func parseUrl(path string) (uint, uint, string, string) {
	w, h := getWidthHeight(path)
	k, b := getKeyBucket(path)
	return w, h, k, b
}

func getKeyBucket(path string) (string, string) {
	ps := strings.Split(path, "/")
	k := strings.Join(ps[3:], "/")
	b := ps[2]
	return b, k
}

func getWidthHeight(path string) (uint, uint) {
	cross := regexp.MustCompile("^\\/(\\d+)(x(\\d+))?\\/").FindAllString(path, -1)
	var width, height uint
	if len(cross) > 0 {
		numbers := regexp.MustCompile("(\\d+)(x(\\d+))?").FindAllStringSubmatch(cross[0], -1)
		if len(numbers[0]) == 4 && numbers[0][3] != "" {
			w, err := strconv.ParseUint(numbers[0][1], 10, 32)
			catchError(err)
			width = uint(w)
			h, err := strconv.ParseUint(numbers[0][3], 10, 32)
			catchError(err)
			height = uint(h)
		} else {
			w, err := strconv.ParseUint(numbers[0][0], 10, 32)
			catchError(err)
			width = uint(w)
		}
	}
	return width, height
}

func getFromS3(bucket, key string) io.Reader {
	var b []byte
	c := aws.NewWriteAtBuffer(b)
	downloader := s3manager.NewDownloader(session.New(&aws.Config{Region: aws.String("ap-southeast-1")}))
	_, err := downloader.Download(c,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
	if err != nil {
		fmt.Println("Failed to download file", err)
	}
	r, err := gzip.NewReader(bytes.NewReader(c.Bytes()))
	if err != nil {
		fmt.Println("Failed to gunzip file", err)
	}
	return r
}

func handler(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	fmt.Printf("Request: %v\n", p)
	if len(strings.Split(p, "/")) > 3 {
		width, height, bucket, key := parseUrl(p)
		img, s, err := image.Decode(getFromS3(bucket, key))
		catchError(err)
		fmt.Println(s)
		m := resize.Resize(width, height, img, resize.Lanczos3)
		fmt.Printf("Delivering: %v/%v -- %vx%v\n", bucket, key, width, height)
		writeImage(w, &m)
	} else {
		fmt.Fprintf(w, "Missing some arguments, %q", html.EscapeString(p))
	}
}

func writeImage(w http.ResponseWriter, img *image.Image) {
	buffer := new(bytes.Buffer)
	if err := jpeg.Encode(buffer, *img, nil); err != nil {
		log.Println("unable to encode image.")
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", strconv.Itoa(len(buffer.Bytes())))
	w.Header().Set("Cache-Control", "max-age=86400, public")
	ETag := sha256.Sum256(buffer.Bytes())
	w.Header().Set("ETag", fmt.Sprintf("%v", ETag))
	if _, err := w.Write(buffer.Bytes()); err != nil {
		log.Println("unable to write image.")
	}
}

func main() {
	fmt.Printf("Server ImagX listening port %v\n", PORT)
	http.HandleFunc("/", handler)
	http.ListenAndServe(fmt.Sprintf(":%v", PORT), nil)
}
