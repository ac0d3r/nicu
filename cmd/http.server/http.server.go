package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"

	"github.com/ac0d3r/nicu/pkg/network"
)

var (
	flagHelp = flag.Bool("h", false, "Shows usage options.")
	flagHost = flag.String("l", ":8080", "listen host")
	flagDir  = flag.String("d", "./", "load directory")
	flagFile = flag.String("f", "", "load single file")
)

func banner() {
	t := `
   __   __  __                                 
  / /  / /_/ /____    ___ ___ _____  _____ ____
 / _ \/ __/ __/ _ \_ (_-</ -_) __/ |/ / -_) __/
/_//_/\__/\__/ .__(_)___/\__/_/  |___/\__/_/   
            /_/                                
`
	fmt.Println(t)
}

func getStatusCode(w http.ResponseWriter) int64 {
	respValue := reflect.ValueOf(w)
	if respValue.Kind() == reflect.Ptr {
		respValue = respValue.Elem()
	}
	status := respValue.FieldByName("status")
	return status.Int()
}

func withlog(h http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
		log.Printf("handle %s %d\n", r.URL.Path, getStatusCode(w))
	})
}

func main() {
	banner()
	flag.Parse()
	if *flagHelp {
		fmt.Printf("Usage: http.server [options]\n\n")
		flag.PrintDefaults()
		return
	}

	if len(strings.Split(*flagHost, ":")[0]) == 0 {
		fmt.Printf("listen on http://localhost%s\n", *flagHost)
		if ipnets, err := network.GetLocalIPV4Net(); err == nil && len(ipnets) >= 1 {
			fmt.Printf("listen on http://%s%s\n", ipnets[0].IP, *flagHost)
		}
	} else {
		fmt.Printf("listen on http://%s\n", *flagHost)
	}

	var handler http.Handler
	if *flagFile != "" {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, *flagFile)
		})
	} else {
		if *flagDir == "" {
			*flagDir = "./"
		}
		handler = http.FileServer(http.Dir(*flagDir))
	}

	log.Fatal(http.ListenAndServe(*flagHost, withlog(handler)))
}
