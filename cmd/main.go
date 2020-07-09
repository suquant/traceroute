package main

import (
	"context"
	"fmt"
	"github.com/suquant/traceroute"
	"log"
	"net"
	"time"
)

func main()  {
	opts := traceroute.Options{
		MaxTTL: 5,
		WaitTime: time.Second * 1,
	}

	dst := net.IPAddr{IP: net.IPv4(8,8,8,8)}
	result, err := traceroute.Traceroute(context.Background(), dst, opts)
	if err != nil {
		log.Fatalf("error: %s", err)
	}

	fmt.Printf("%+v", result)
}