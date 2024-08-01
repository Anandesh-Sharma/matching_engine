package main

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	socketio "github.com/googollee/go-socket.io"
)

type Order struct {
	UserID    string  `json:"user_id"`
	Asset     string  `json:"asset"`
	OrderType string  `json:"order_type"`
	Price     float64 `json:"price"`
	Amount    int     `json:"amount"`
}

type OrderBook struct {
	BuyOrders  OrderHeap
	SellOrders OrderHeap
}

type OrderHeap []*Order

func (h OrderHeap) Len() int { return len(h) }
func (h OrderHeap) Less(i, j int) bool {
	if h[i].OrderType == "buy" {
		return h[i].Price > h[j].Price // Max heap for buy orders
	}
	return h[i].Price < h[j].Price // Min heap for sell orders
}
func (h OrderHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *OrderHeap) Push(x interface{}) { *h = append(*h, x.(*Order)) }
func (h *OrderHeap) Pop() interface{} {
	old := *h
	n := len(old)
	order := old[n-1]
	*h = old[0 : n-1]
	return order
}

type MatchingEngine struct {
	OrderBooks map[string]*OrderBook
	Lock       sync.Mutex
}

func NewMatchingEngine() *MatchingEngine {
	return &MatchingEngine{
		OrderBooks: make(map[string]*OrderBook),
	}
}

func (engine *MatchingEngine) AddOrder(order *Order, s socketio.Conn) {
	engine.Lock.Lock()
	defer engine.Lock.Unlock()

	if _, exists := engine.OrderBooks[order.Asset]; !exists {
		engine.OrderBooks[order.Asset] = &OrderBook{}
	}

	orderBook := engine.OrderBooks[order.Asset]
	if order.OrderType == "buy" {
		heap.Push(&orderBook.BuyOrders, order)
	} else {
		heap.Push(&orderBook.SellOrders, order)
	}
	engine.MatchOrders(orderBook, s)
}

func (engine *MatchingEngine) MatchOrders(orderBook *OrderBook, s socketio.Conn) {
	for len(orderBook.BuyOrders) > 0 && len(orderBook.SellOrders) > 0 {
		bestBuy := orderBook.BuyOrders[0]
		bestSell := orderBook.SellOrders[0]

		if bestBuy.Price >= bestSell.Price {
			matchedAmount := min(bestBuy.Amount, bestSell.Amount)
			fmt.Printf("Matching %d of %s (buy) with %s (sell) at price %.2f\n", matchedAmount, bestBuy.UserID, bestSell.UserID, bestSell.Price)

			// Broadcast the matched order
			matchedData := map[string]interface{}{
				"status":         "success",
				"matched_amount": matchedAmount,
				"buyer":          bestBuy.UserID,
				"seller":         bestSell.UserID,
				"price":          bestSell.Price,
			}
			s.Emit("order_matched", matchedData)

			bestBuy.Amount -= matchedAmount
			bestSell.Amount -= matchedAmount

			if bestBuy.Amount == 0 {
				heap.Pop(&orderBook.BuyOrders)
			}
			if bestSell.Amount == 0 {
				heap.Pop(&orderBook.SellOrders)
			}
		} else {
			break
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var engine = NewMatchingEngine()

var socketServer *socketio.Server

func main() {
	socketServer = socketio.NewServer(nil)

	socketServer.OnConnect("/", func(s socketio.Conn) error {
		fmt.Println("Client connected:", s.ID())
		return nil
	})

	socketServer.OnDisconnect("/", func(s socketio.Conn, msg string) {
		fmt.Println("Client disconnected:", s.ID())
	})

	socketServer.OnEvent("/", "new_order", func(s socketio.Conn, data string) {
		var order Order
		if err := json.Unmarshal([]byte(data), &order); err != nil {
			fmt.Println("Error unmarshalling order:", err)
			return
		}
		engine.AddOrder(&order, s)
		s.Emit("order_received", map[string]string{"status": "success"})
	})

	http.Handle("/socket.io/", socketServer)
	http.Handle("/", http.FileServer(http.Dir("./static"))) // Serve static files if needed

	go func() {
		if err := socketServer.Serve(); err != nil {
			fmt.Println("Socket server error:", err)
		}
	}()
	defer socketServer.Close()

	fmt.Println("Server started at :5001")
	if err := http.ListenAndServe(":5001", nil); err != nil {
		fmt.Println("HTTP server error:", err)
	}
}
