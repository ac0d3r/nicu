package main

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/ac0d3r/nicu/pkg/network"
	"github.com/ac0d3r/nicu/pkg/pingx"
)

func banner() {
	t := `
     .               
 ___    ___  ___      
|   )| |   )|   )(_/_ 
|__/ | |  / |__/  / / 
|           __/       
`
	fmt.Println(t)
}

func main() {
	banner()
	ipnet, err := network.GetLocalIPV4Net()
	if err != nil || len(ipnet) == 0 {
		log.Fatal("get local ipv4 IPNet fail: ", err)
	}
	fmt.Println("IPNet list: ")
	for i := range ipnet {
		fmt.Printf("\t%d: %s/%s \n", i+1, ipnet[i].IP.String(), ipnet[i].Mask.String())
	}

	var input string
	fmt.Print("select IPNet: ")
	fmt.Scanln(&input)

	index, err := strconv.ParseInt(input, 10, 64)
	index--
	if err != nil || index < 0 || int(index) >= len(ipnet) {
		log.Fatal("select IPNet fail: ", err)
	}

	pingxer, err := pingx.NewPingxer()
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result, err := pingxer.Scan(ctx, ipnet[index])
	if err != nil {
		log.Fatal(err)
	}
	for i := range result {
		fmt.Println("[+]", result[i])
	}
}
