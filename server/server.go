package server

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"exchanger/orderbook"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

const (
	MarketOrder OrderType = "MARKET"
	LimitOrder  OrderType = "LIMIT"

	MarketETH Market = "ETH"
)

type (
	OrderType string
	Market    string
)

type PlaceOrderRequest struct {
	UserID int64
	Type   OrderType // limit or market
	Bid    bool
	Size   float64
	Price  float64
	Market Market
}

type Order struct {
	UserID    int64
	ID        int64
	Price     float64
	Size      float64
	Bid       bool
	Timestamp int64
}

type OrderbookData struct {
	TotalBidVolume float64
	TotalAskVolume float64
	Asks           []*Order
	Bids           []*Order
}

func StartServer() {
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler

	client, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		log.Fatal(err)
	}

	ex, err := NewExchanger(client)
	if err != nil {
		log.Fatal(err)
	}
	pkStr8 := "829e924fdf021ba3dbbc4225edfece9aca04b929d6e75613329ca6f1d31c0bb4"
	user8 := NewUser(pkStr8, 8)
	ex.Users[user8.ID] = user8

	pkStr7 := "a453611d9419d0e56f499079478fd72c37b251a94bfde4d19872c44cf65386e3"
	user7 := NewUser(pkStr7, 7)
	ex.Users[user7.ID] = user7

	e.GET("/order/:userID", ex.handleGetOrders)
	e.GET("/book/:market", ex.handleGetBook)
	e.POST("/order", ex.handlePlaceOrder)
	e.DELETE("/order/:id", ex.cancelOrder)
	e.GET("/book/:market/bid", ex.handleGetBestBid)
	e.GET("/book/:market/ask", ex.handleGetBestAsk)

	sellerAddress := "0xACa94ef8bD5ffEE41947b4585a84BdA5a3d3DA6E"
	sellerBalance, _ := ex.Client.BalanceAt(context.Background(), common.HexToAddress(sellerAddress), nil)
	fmt.Printf("[seller balance: %v\n", sellerBalance)

	buyerAddress := "0x28a8746e75304c0780E011BEd21C72cD78cd535E"
	buyerBalance, _ := ex.Client.BalanceAt(context.Background(), common.HexToAddress(buyerAddress), nil)
	fmt.Printf("[buyer balance: %v\n", buyerBalance)

	e.Start(":3000")
}

type User struct {
	ID         int64
	PrivateKey *ecdsa.PrivateKey
}

func NewUser(pk string, id int64) *User {
	privateKey, err := crypto.HexToECDSA(pk)
	if err != nil {
		panic(err)
	}
	return &User{
		ID:         id,
		PrivateKey: privateKey,
	}
}

func httpErrorHandler(err error, c echo.Context) {
	fmt.Println(err)
}

type Exchange struct {
	Client *ethclient.Client
	mu     sync.RWMutex
	Users  map[int64]*User
	// Orders maps a user to his order
	Orders     map[int64][]*orderbook.Order
	PrivateKey *ecdsa.PrivateKey
	orderbooks map[Market]*orderbook.Orderbook
}

func NewExchanger(client *ethclient.Client) (*Exchange, error) {
	exchangerPrivateKey := privateKeyFromENV()
	orderbooks := make(map[Market]*orderbook.Orderbook)
	orderbooks[MarketETH] = orderbook.NewOrderbook()
	privateKey, err := crypto.HexToECDSA(exchangerPrivateKey)
	if err != nil {
		return nil, err
	}
	return &Exchange{
		Client:     client,
		Users:      make(map[int64]*User),
		Orders:     make(map[int64][]*orderbook.Order),
		PrivateKey: privateKey,
		orderbooks: orderbooks,
	}, nil
}

type GetOrdersResponce struct {
	Asks []Order
	Bids []Order
}

func (ex *Exchange) handleGetOrders(c echo.Context) error {
	userIDStr := c.Param("userID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return err
	}

	ex.mu.RLock()
	orderbookOrders := ex.Orders[int64(userID)]
	orderResponce := &GetOrdersResponce{
		Asks: []Order{},
		Bids: []Order{},
	}

	for i := 0; i < len(orderbookOrders); i++ {
		order := Order{
			ID:        orderbookOrders[i].ID,
			UserID:    orderbookOrders[i].UserID,
			Price:     orderbookOrders[i].Limit.Price,
			Size:      orderbookOrders[i].Size,
			Timestamp: orderbookOrders[i].Timestamp,
			Bid:       orderbookOrders[i].Bid,
		}

		if order.Bid {
			orderResponce.Bids = append(orderResponce.Bids, order)
		} else {
			orderResponce.Asks = append(orderResponce.Asks, order)
		}
	}
	ex.mu.RUnlock()
	return c.JSON(http.StatusOK, orderResponce)
}

func (ex *Exchange) handleGetBook(c echo.Context) error {
	market := Market(c.Param("market"))
	ob, ok := ex.orderbooks[market]
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]any{"msg": "market not found"})
	}

	orderbookData := OrderbookData{
		TotalBidVolume: ob.BidTotalVolume(),
		TotalAskVolume: ob.AskTotalVolume(),
		Asks:           []*Order{},
		Bids:           []*Order{},
	}

	for _, limit := range ob.Asks() {
		for _, order := range limit.Orders {
			o := Order{
				UserID:    order.UserID,
				ID:        order.ID,
				Price:     limit.Price,
				Size:      order.Size,
				Bid:       order.Bid,
				Timestamp: order.Timestamp,
			}
			orderbookData.Asks = append(orderbookData.Asks, &o)
		}
	}
	for _, limit := range ob.Bids() {
		for _, order := range limit.Orders {
			o := Order{
				UserID:    order.UserID,
				ID:        order.ID,
				Price:     limit.Price,
				Size:      order.Size,
				Bid:       order.Bid,
				Timestamp: order.Timestamp,
			}
			orderbookData.Bids = append(orderbookData.Bids, &o)
		}
	}
	return c.JSON(http.StatusOK, orderbookData)
}

type PriceResponce struct {
	Price float64
}

func (ex *Exchange) handleGetBestBid(c echo.Context) error {
	market := Market(c.Param("market"))
	ob := ex.orderbooks[market]
	if len(ob.Bids()) == 0 {
		return fmt.Errorf("the bids are empty")
	}
	bestBidPrice := ob.Bids()[0].Price
	pr := PriceResponce{
		Price: bestBidPrice,
	}
	return c.JSON(http.StatusOK, pr)

}

func (ex *Exchange) handleGetBestAsk(c echo.Context) error {
	market := Market(c.Param("market"))
	ob := ex.orderbooks[market]
	if len(ob.Asks()) == 0 {
		return fmt.Errorf("the asks are empty")
	}
	bestAskPrice := ob.Asks()[0].Price
	pr := PriceResponce{
		Price: bestAskPrice,
	}
	return c.JSON(http.StatusOK, pr)

}

func (ex *Exchange) cancelOrder(c echo.Context) error {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)

	ob := ex.orderbooks[MarketETH]
	order := ob.Orders[int64(id)]
	ob.CancelOrder(order)

	log.Println("order canceled id -> ", id)

	return c.JSON(200, map[string]any{"msg": "order deleted"})
}

type MatchedOrder struct {
	UserID int64
	Price  float64
	Size   float64
	ID     int64
}

func (ex *Exchange) handlePlaceMarketOrder(market Market, order *orderbook.Order) ([]orderbook.Match, []*MatchedOrder) {
	ob := ex.orderbooks[market]
	matches := ob.PlaceMarketOrder(order)
	matchedOrders := make([]*MatchedOrder, len(matches))

	isBid := false
	if order.Bid {
		isBid = true
	}
	totalSizeFilled := 0.0
	for i := 0; i < len(matchedOrders); i++ {
		limitUserID := matches[i].Bid.UserID
		id := matches[i].Bid.ID
		if isBid {
			limitUserID = matches[i].Ask.UserID
			id = matches[i].Ask.ID
		}
		matchedOrders[i] = &MatchedOrder{
			UserID: limitUserID,
			ID:     id,
			Size:   matches[i].SizeFilled,
			Price:  matches[i].Price,
		}
		totalSizeFilled += matches[i].SizeFilled
	}
	avgPrice := 0.0
	for i := 0; i < len(matches); i++ {
		avgPrice += matches[i].Price / totalSizeFilled * matches[i].SizeFilled
	}
	log.Printf("filled MARKET order -> %d | size [%.2f] | avgPrice [%.2f]", order.ID, totalSizeFilled, avgPrice)

	newOrderMap := make(map[int64][]*orderbook.Order)
	ex.mu.Lock()
	for userID, orderbookOrders := range ex.Orders {
		for i := 0; i < len(orderbookOrders); i++ {
			// if the order is not filled we place it in the map copy. Size of the order = 0.
			if !orderbookOrders[i].IsFilled() {
				newOrderMap[userID] = append(newOrderMap[userID], orderbookOrders[i])

			}
		}
	}
	ex.Orders = newOrderMap
	ex.mu.Unlock()

	return matches, matchedOrders
}

func (ex *Exchange) handlePlaceLimitOrder(market Market, price float64, order *orderbook.Order) error {
	ob := ex.orderbooks[market]
	ob.PlaceLimitOrder(price, order)

	ex.mu.Lock()
	ex.Orders[order.UserID] = append(ex.Orders[order.UserID], order)
	ex.mu.Unlock()

	log.Printf("new LIMIT order -> type: [%t] | price [%.2f | size [%.2f]", order.Bid, order.Limit.Price, order.Size)

	return nil
}

type PlaceOrderResponce struct {
	OrderID int64
}

func (ex *Exchange) handlePlaceOrder(c echo.Context) error {
	var placeOrderData PlaceOrderRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&placeOrderData); err != nil {
		return err
	}

	market := placeOrderData.Market
	order := orderbook.NewOrder(placeOrderData.Bid, placeOrderData.Size, placeOrderData.UserID)

	if placeOrderData.Type == LimitOrder {
		if err := ex.handlePlaceLimitOrder(market, placeOrderData.Price, order); err != nil {
			return err
		}
	}

	if placeOrderData.Type == MarketOrder {
		matches, _ := ex.handlePlaceMarketOrder(market, order)
		if err := ex.handleMatches(matches); err != nil {
			return err
		}

	}

	resp := &PlaceOrderResponce{
		OrderID: order.ID,
	}
	return c.JSON(200, resp)
}

func (ex *Exchange) handleMatches(matches []orderbook.Match) error {
	for _, match := range matches {
		fromUser, ok := ex.Users[match.Ask.UserID]
		if !ok {
			return fmt.Errorf("user not found: %d", match.Ask.UserID)
		}
		toUser, ok := ex.Users[match.Bid.UserID]
		if !ok {
			return fmt.Errorf("user not found: %d", match.Bid.UserID)
		}
		toAddress := crypto.PubkeyToAddress(toUser.PrivateKey.PublicKey)

		// for the fees
		// exchangePublicKey := ex.PrivateKey.Public()
		// publicKeyECDSA, ok := exchangePublicKey.(*ecdsa.PublicKey)
		// if !ok {
		// 	return fmt.Errorf("error casting public key to ECDSA")
		// }

		amount := big.NewInt(int64(match.SizeFilled))
		transferETH(ex.Client, fromUser.PrivateKey, toAddress, amount)
	}
	return nil
}

func privateKeyFromENV() (exchangerPrivateKey string) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	exchangerPrivateKey = os.Getenv("EX_PRIVATE_KEY")
	return exchangerPrivateKey
}
