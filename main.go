package main

import (
	"exchanger/client"
	"exchanger/server"
	"fmt"
	"time"
)

var tick = 1 * time.Second

func makeMarketSimple(cleint *client.Client) {
	ticker := time.NewTicker(tick)

	for {
		<-ticker.C
		fmt.Println("test")
	}
}

func seedMarket(c *client.Client) error {
	ask := &client.PlaceOrderParams{
		UserID: 8,
		Bid:    false,
		Price:  10_000,
		Size:   1_000_000,
	}
	bid := &client.PlaceOrderParams{
		UserID: 8,
		Bid:    true,
		Price:  9_000,
		Size:   1_000_000,
	}

	_, err := c.PlaceLimitOrder(ask)
	if err != nil {
		return err
	}
	_, err = c.PlaceLimitOrder(bid)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	go server.StartServer()

	time.Sleep(1 * time.Second)

	c := client.NewClient()

	if err := seedMarket(c); err != nil {
		panic(err)
	}

	makeMarketSimple(c)

	// for {
	// 	limitOrderParams := &client.PlaceOrderParams{
	// 		UserID: 8,
	// 		Bid:    true,
	// 		Price:  10_000,
	// 		Size:   320,
	// 	}
	// 	_, err := c.PlaceLimitOrder(limitOrderParams)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	// fmt.Println("placed limit order from the client -> ", resp.OrderID)

	// 	otherLimitOrderParams := &client.PlaceOrderParams{
	// 		UserID: 8,
	// 		Bid:    false,
	// 		Price:  7_000,
	// 		Size:   680,
	// 	}
	// 	_, err = c.PlaceLimitOrder(otherLimitOrderParams)
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	// marketOrderParams := &client.PlaceOrderParams{
	// 	// 	UserID: 7,
	// 	// 	Bid:    true,
	// 	// 	Price:  10_000,
	// 	// 	Size:   1000,
	// 	// }
	// 	// _, err = c.PlaceMarketOrder(marketOrderParams)
	// 	// if err != nil {
	// 	// 	panic(err)
	// 	// }
	// 	// fmt.Println("placed market order from the client -> ", resp.OrderID)

	// 	time.Sleep(1 * time.Second)

	// }

	select {}
}
