package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	// grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/odit-bit/scarlett/store/storepb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func generateKeyValueData(dur time.Duration, size int, addr string) {
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGINT)

	//retry
	retryOpts := []grpc_retry.CallOption{
		grpc_retry.WithBackoff(grpc_retry.BackoffExponential(100 * time.Millisecond)),
		grpc_retry.WithMax(3), // Give up after 5 retries.
	}
	retry := grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(retryOpts...))

	// credentials
	creds := grpc.WithTransportCredentials(insecure.NewCredentials())

	// client instance
	// log.Println(addr)
	conn, err := grpc.NewClient(
		addr,
		creds,
		retry,
	)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()
	fmt.Println(conn.GetState().String())

	// exec command
	cli := storepb.NewCommandClient(conn)

	// data := bytes.Repeat([]byte{'x'}, size)
	count := atomic.Int64{}
	keys := "fizz"
	timer := time.NewTimer(dur)

	pref := 0
	key := fmt.Sprintf("%s-%v", keys, pref)
	// value := fmt.Sprintf("%s-%v", data, pref)
	req := storepb.SetRequest{
		Cmd:   storepb.Command_Type_Set,
		Key:   []byte(key),
		Value: []byte(time.Now().String()),
	}
	for {

		select {
		case <-sigC:
			log.Println("got signal")
		case <-timer.C:
			timer.Stop()
			log.Println("finish")
		default:
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_, err := cli.Set(ctx, &req)
			cancel()
			if err == nil {
				count.Add(1)
				continue
			}
			log.Println(err)
		}
		break
	}

	close(sigC)
	log.Println("total-count", count.Load())

}

func main() {
	dur, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	seconds := time.Duration(dur) * time.Second
	fmt.Println("run for", seconds)
	generateKeyValueData(seconds, 1024, os.Args[1])
}
