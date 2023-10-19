package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"sync"

	"github.com/quic-go/quic-go"
)

type MessageBroker struct {
	publisherPort  int
	subscriberPort int
	subscribers    map[quic.Stream]bool
	subscribersMu  sync.RWMutex
}

func NewMessageBroker(publisherPort, subscriberPort int) *MessageBroker {
	return &MessageBroker{
		publisherPort:  publisherPort,
		subscriberPort: subscriberPort,
		subscribers:    make(map[quic.Stream]bool),
	}
}

func (mb *MessageBroker) Start() {
	go mb.listenPublisher()
	go mb.listenSubscriber()
}

func (mb *MessageBroker) notifyPublisher(stream quic.Stream) {
	mb.subscribersMu.RLock()
	defer mb.subscribersMu.RUnlock()

	if len(mb.subscribers) == 0 {
		_, err := fmt.Fprintf(stream, "No subscribers connected\n")
		if err != nil {
			log.Println("Error notifying publisher:", err)
		}
		fmt.Println("No subscribers connected")
	} else {
		_, err := fmt.Fprintf(stream, "Subscriber connected\n")
		if err != nil {
			log.Println("Error notifying publisher:", err)
		}
		fmt.Println("Subscriber connected")

		go mb.broadcastMessage("New subscriber connected")
	}
}

func (mb *MessageBroker) broadcastMessage(message string) {
	mb.subscribersMu.RLock()
	defer mb.subscribersMu.RUnlock()

	for stream := range mb.subscribers {
		_, err := fmt.Fprintf(stream, "Message: %s\n", message)
		if err != nil {
			log.Println("Error broadcasting message:", err)
		}
	}
	fmt.Printf("Broadcasted message: %s\n", message)
}

func (mb *MessageBroker) listenPublisher() {
	listener, err := quic.ListenAddr(fmt.Sprintf(":%d", mb.publisherPort), &tls.Config{}, nil)
	if err != nil {
		log.Fatal(err)
	}

	for {
		sess, err := listener.Accept(context.Background())
		if err != nil {
			log.Fatal(err)
		}

		stream, err := sess.AcceptStream(context.Background())
		if err != nil {
			log.Fatal(err)
		}

		go func(stream quic.Stream) {
			defer stream.Close()

			fmt.Println("Publisher connected")
			mb.notifyPublisher(stream)
		}(stream)
	}
}

func (mb *MessageBroker) listenSubscriber() {
	listener, err := quic.ListenAddr(fmt.Sprintf(":%d", mb.subscriberPort), &tls.Config{}, nil)
	if err != nil {
		log.Fatal(err)
	}

	for {
		sess, err := listener.Accept(context.Background())
		if err != nil {
			log.Fatal(err)
		}

		stream, err := sess.AcceptStream(context.Background())
		if err != nil {
			log.Fatal(err)
		}

		mb.subscribersMu.Lock()
		mb.subscribers[stream] = true
		mb.subscribersMu.Unlock()

		fmt.Println("Subscriber connected")
	}
}

func main() {
	messageBroker := NewMessageBroker(5000, 5001)
	messageBroker.Start()

	select {}
}
