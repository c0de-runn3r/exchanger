package main

import (
	"exchanger/client"
	"exchanger/server"
	"fmt"
	"log"
	"math"
	"time"
)

const (
	maxOrders = 3
)

var (
	tick = 2 * time.Second
)

func marketOrderPlacer(c *client.Client) {
	ticker := time.NewTicker(tick)
	for {
		trades, err := c.GetTrades("ETH")
		if err != nil {
			panic(err)
		}
		if len(trades) > 0 {
			fmt.Printf("LAST TRADE PRICE -> %.2f\n", trades[len(trades)-1].Price)
		}

		marketSellOrder := &client.PlaceOrderParams{
			UserID: 8,
			Bid:    false,
			Size:   3000,
		}
		sellOrderResp, err := c.PlaceMarketOrder(marketSellOrder)
		if err != nil {
			log.Printf("error: %v ", sellOrderResp.OrderID)
		}
		marketBuyOrder := &client.PlaceOrderParams{
			UserID: 8,
			Bid:    true,
			Size:   900,
		}
		buyOrderResp, err := c.PlaceMarketOrder(marketBuyOrder)
		if err != nil {
			log.Printf("error: %v ", buyOrderResp.OrderID)
		}

		<-ticker.C
	}
}

const userID = 7

func makeMarketSimple(c *client.Client) {
	ticker := time.NewTicker(tick)
	for {
		orders, err := c.GetOrders(userID)
		if err != nil {
			log.Println(err)
		}
		fmt.Printf("%+v\n", orders)

		bestAsk, err := c.GetBestAsk()
		if err != nil {
			log.Println(err)
		}
		bestBid, err := c.GetBestBid()
		if err != nil {
			log.Println(err)
		}
		spread := math.Abs(bestBid - bestAsk)
		fmt.Println("exchange spread ", spread)
		// place the bid
		if len(orders.Bids) < maxOrders {

			bidLimit := &client.PlaceOrderParams{
				UserID: 7,
				Bid:    true,
				Price:  bestBid + 100,
				Size:   1000,
			}

			bidOrderResp, err := c.PlaceLimitOrder(bidLimit)
			if err != nil {
				log.Printf("error: %v ", bidOrderResp.OrderID)
			}
		}
		// place the ask
		if len(orders.Asks) < maxOrders {
			askLimit := &client.PlaceOrderParams{
				UserID: 7,
				Bid:    false,
				Price:  bestAsk - 100,
				Size:   1000,
			}
			askOrderResp, err := c.PlaceLimitOrder(askLimit)
			if err != nil {
				log.Printf("error: %v ", askOrderResp.OrderID)
			}
		}
		fmt.Println("best ask price ", bestAsk)
		fmt.Println("best bid price ", bestBid)

		<-ticker.C
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

	go makeMarketSimple(c)
	time.Sleep(1 * time.Second)
	marketOrderPlacer(c)

	select {}
}
