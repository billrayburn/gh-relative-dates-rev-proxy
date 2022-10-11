package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var dateRegex = regexp.MustCompile(`(created|closed):[<>]([0-9+dmh]+)`)

// Get env var or default
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// Get the port to listen on
func getListenAddress() string {
	port := getEnv("PORT", "1338")
	return ":" + port
}

// Log the env variables required for a reverse proxy
func logSetup() {
	log.Printf("Server will run on: %s\n", getListenAddress())
}

func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {
	// parse the url
	targetURL, _ := url.Parse("https://github.com")
	log.Printf("redirecting to %s", req.URL)
	qvals, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		log.Printf("unable to unescape query string %s: %v", req.URL.RawQuery, err)
	} else if qvals.Get("q") != "" {
		decodedValue := qvals.Get("q")
		submatches := dateRegex.FindStringSubmatch(decodedValue)
		if len(submatches) > 2 {
			for _, candidateDuration := range submatches[2:] {
				ogDuration := candidateDuration
				if strings.HasSuffix(candidateDuration, "d") {
					strippedDuration := strings.ReplaceAll(candidateDuration, "d", "")
					iDur, err := strconv.Atoi(strippedDuration)
					if err != nil {
						log.Printf("unable to convert duration %s to int: %v", candidateDuration, err)
						continue
					}
					candidateDuration = fmt.Sprintf("%dh", iDur*24)
				}
				duration, err := time.ParseDuration(candidateDuration)
				if err != nil {
					log.Printf("unable to parse duration %s", candidateDuration)
					continue
				}
				decodedValue = strings.ReplaceAll(decodedValue, ogDuration, time.Now().Add(-1*duration).Format("2006-01-02"))
				log.Printf("rewrote query string to %s", decodedValue)
			}
			qvals.Set("q", decodedValue)
			req.URL.RawQuery = qvals.Encode()
		}
	}

	// Update the headers to allow for SSL redirection
	req.URL.Host = targetURL.Host
	req.URL.Scheme = targetURL.Scheme
	req.Host = targetURL.Host

	log.Printf("redirecting rewritten req to %s", req.URL)
	http.Redirect(res, req, req.URL.String(), http.StatusTemporaryRedirect)
}

func main() {
	logSetup()

	// start server
	http.HandleFunc("/", handleRequestAndRedirect)
	if err := http.ListenAndServe(getListenAddress(), nil); err != nil {
		panic(err)
	}
}
