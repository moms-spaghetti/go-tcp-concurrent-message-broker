package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

const (
	network = "tcp"
	address = ":9000"
)

var (
	subscribe = make(chan request)
	publish   = make(chan publishReq)
)

type request struct {
	Action  string `json:"action"`
	Topic   string `json:"topic"`
	data    chan string
	message chan string
}

type response struct {
	Message string
}

type topics map[string][]chan string

type publishReq struct {
	topic   string
	message string
}

func main() {
	go topicMonitor()
	go publishMessage()

	laddr, err := net.ResolveTCPAddr(network, address)
	if err != nil {
		panic(err)
	}

	lis, err := net.ListenTCP(network, laddr)
	if err != nil {
		panic(err)
	}
	log.Print("tcp listening...")

	for {
		conn, err := lis.AcceptTCP()
		if err != nil {
			panic(err)
		}

		go newConn(conn)
	}
}

func newConn(conn *net.TCPConn) {
	req := request{
		data:    make(chan string),
		message: make(chan string),
	}

	go func() {
		for m := range req.message {
			var res = response{
				Message: m,
			}
			if err := json.NewEncoder(conn).Encode(res); err != nil {
				panic(err)
			}
		}
	}()

	for {
		if err := json.NewDecoder(conn).Decode(&req); err != nil {
			panic(err)
		}

		switch req.Action {
		case "quit":
			conn.Close()
			return
		case "sub":
			subscribe <- req
		}

		var res = response{
			Message: <-req.data,
		}
		if err := json.NewEncoder(conn).Encode(res); err != nil {
			panic(err)
		}
	}
}

func topicMonitor() {
	token := make(chan struct{}, 1)
	token <- struct{}{}

	t := topics{}

	for {
		select {
		case req := <-subscribe:
			<-token
			t[req.Topic] = append(t[req.Topic], req.message)
			req.data <- fmt.Sprintf("subscribed to %s", req.Topic)
			token <- struct{}{}
		case m := <-publish:
			<-token
			subs := t[m.topic]
			for _, sc := range subs {
				sc <- m.message
			}
			token <- struct{}{}
		}
	}
}

//publish
func publishMessage() {
	time.Sleep(50 * time.Millisecond)
	for {
		topic := readEntry("topic")
		message := readEntry("message")
		publish <- publishReq{
			topic:   topic,
			message: message,
		}
	}
}

func readEntry(instruction string) string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print(instruction, "-> ")
	entry, _ := reader.ReadString('\n')
	return strings.Replace(entry, "\n", "", -1)
}
