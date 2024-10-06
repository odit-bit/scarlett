package main

import (
	"context"
	"crypto/rand"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/odit-bit/scarlett/api/cluster"
)

func generateKeyValueData(n, size int, addr string) {
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGINT)

	cli, err := cluster.NewClient()
	if err != nil {
		panic(err)
	}

	data := make([]byte, size)

	count := -1
	keys := ""
	errCount := 0
	for count < n {
		count++
		select {
		case <-sigC:
			log.Println("got signal, shutdown")
			return
		default:
		}

		if count%500 == 0 {
			log.Println("current :", count)
		}
		if count%2 == 0 {
			keys = "fizz"
		} else if count%3 == 0 {
			keys = "buzz"
		} else {
			keys = "fizzbuzz"
		}

		if _, err := rand.Read(data); err != nil {
			panic(err)
		}
		ctx, cancel := context.WithTimeout(context.TODO(), 2*time.Second)
		_, err := cli.Set(ctx, addr, keys, string(data))
		if err != nil {
			errCount++
			if ctx.Err() != nil {
				log.Println(ctx.Err())
				continue
			}
			log.Fatalf("error %v, type %T", err, err)
		}
		cancel()
		// time.Sleep(1 * time.Millisecond)
	}
	// _, err = cli.Set(context.TODO(), addr, "ulala", string(data))
	// if err != nil {
	// 	panic(err)
	// }

	close(sigC)
	<-sigC
	log.Println("err-count", errCount)
}

func main() {
	generateKeyValueData(10000, 4096, os.Args[1])
}
