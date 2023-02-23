package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	network = "tcp"
	address = ":9000"
)

type request struct {
	Action string
	Topic  string
}

type response struct {
	Message string
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	wait := make(chan os.Signal, 1)
	signal.Notify(wait, syscall.SIGINT)

	raddr, err := net.ResolveTCPAddr(network, address)
	if err != nil {
		panic(err)
	}

	conn, err := net.DialTCP(network, nil, raddr)
	if err != nil {
		panic(err)
	}

	fmt.Printf("client: %s\n", conn.LocalAddr().String())

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(50 * time.Millisecond)
				topic := readEntry("topic")
				action := readEntry("action")

				var req = request{
					Action: action,
					Topic:  topic,
				}

				if err := json.NewEncoder(conn).Encode(req); err != nil {
					panic(err)
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var res response
				if err := json.NewDecoder(conn).Decode(&res); err != nil {
					panic(err)
				}

				fmt.Printf("received: %s\n", res.Message)
			}
		}
	}()

	<-wait
	cancel()
	var req = request{
		Action: "quit",
	}

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		panic(err)
	}
}

func readEntry(instruction string) string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print(instruction, "-> ")
	entry, _ := reader.ReadString('\n')
	return strings.Replace(entry, "\n", "", -1)
}
