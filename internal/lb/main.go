package main

import (
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGINT)

	// listen  TO
	addr, err := net.ResolveTCPAddr("tcp", "localhost:9999")
	if err != nil {
		log.Fatal(err)
	}
	l, err := net.ListenTCP(addr.Network(), addr)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	// accepting
	// run on diffrent thread
	connC := make(chan *net.TCPConn, 1)
	go func() {
		defer func() {
			close(connC)
		}()

		for {
			conn, err := l.AcceptTCP()
			if err != nil {
				log.Println(err)
				break
			}
			connC <- conn
		}
	}()

	for {
		select {
		case conn, ok := <-connC:
			// forward TO
			fAddr, err := net.ResolveTCPAddr("tcp", "localhost:8001")
			if err != nil {
				log.Fatal(err)
			}
			tConn, err := net.DialTCP("tcp", nil, fAddr)
			if err != nil {
				log.Fatal(err)
			}
			defer tConn.Close()
			if ok {
				job := make(chan struct{})
				go func() {
					defer func() {
						conn.Close()
					}()
					log.Printf("forward connection to %s from %s \n", tConn.RemoteAddr(), conn.RemoteAddr())
					go func() {
						if _, err := io.Copy(tConn, conn); err != nil {
							log.Println(err)
						}
						conn.CloseRead()
						close(job)
					}()
					
					go func() {
						if _, err := io.Copy(conn, tConn); err != nil {
							log.Println(err)
						}
					}()
					<-job
					log.Printf("Finish proxieng to %s from %s \n", tConn.RemoteAddr(), conn.RemoteAddr())
				}()
				continue
			}
		case <-sigC:
			l.Close()
			log.Println("got signal")
		}
		break
	}

	log.Println("shutdown")
}
