package main

import (
	"exchanger/client"
	"exchanger/server"
	"time"
)

func main() {
	go server.StartServer()

	time.Sleep(1 * time.Second)

	c := client.NewClient()
	for {
		limitOrderParams := &client.PlaceOrderParams{
			UserID: 8,
			Bid:    false,
			Price:  10_000,
			Size:   320,
		}
		_, err := c.PlaceLimitOrder(limitOrderParams)
		if err != nil {
			panic(err)
		}
		// fmt.Println("placed limit order from the client -> ", resp.OrderID)

		otherLimitOrderParams := &client.PlaceOrderParams{
			UserID: 8,
			Bid:    false,
			Price:  7_000,
			Size:   680,
		}
		_, err = c.PlaceLimitOrder(otherLimitOrderParams)
		if err != nil {
			panic(err)
		}

		marketOrderParams := &client.PlaceOrderParams{
			UserID: 7,
			Bid:    true,
			Price:  10_000,
			Size:   1000,
		}
		_, err = c.PlaceMarketOrder(marketOrderParams)
		if err != nil {
			panic(err)
		}
		// fmt.Println("placed market order from the client -> ", resp.OrderID)

		time.Sleep(1 * time.Second)

	}

	select {}
}
