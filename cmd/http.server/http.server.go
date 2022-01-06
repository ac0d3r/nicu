package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"reflect"
	"strings"
)

var (
	flagHelp = flag.Bool("h", false, "Shows usage options.")
	flagHost = flag.String("l", ":8080", "listen host")
	flagDir  = flag.String("d", "./", "load directory")
	flagFile = flag.String("f", "", "load single file")
)

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
	flag.Parse()
	if *flagHelp {
		fmt.Printf("Usage: http.server [options]\n\n")
		flag.PrintDefaults()
		return
	}

	if len(strings.Split(*flagHost, ":")[0]) == 0 {
		fmt.Printf("listen on http://localhost%s\n", *flagHost)
		if ip, err := getLocalIpV4(); err == nil && ip != "" {
			fmt.Printf("listen on http://%s%s\n", ip, *flagHost)
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

func getLocalIpV4() (string, error) {
	inters, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, inter := range inters {
		if inter.Flags&net.FlagUp != 0 && !strings.HasPrefix(inter.Name, "lo") {
			addrs, err := inter.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						return ipnet.IP.String(), nil
					}
				}
			}
		}
	}
	return "", nil
}
