package bitflyer

import (
	"log"
	"os"
	"os/signal"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mitsutoshi/bitflyergo"
)

var (
	BidTotalSize = 0.0
	AskTotalSize = 0.0
	Health       = ""
	Ltp          float64
	MidPrice     float64
	Bids         = map[float64]float64{}
	Asks         = map[float64]float64{}
	BoardLock    = sync.Mutex{}
	restClient   = bitflyergo.NewBitflyer("", "", []int{-208}, 3, 1)
)

// Returns sorted ask prices.
func SortedAskPrices() []float64 {
	prices := make([]float64, 0, len(Asks))
	for k := range Asks {
		prices = append(prices, k)
	}
	sort.Sort(sort.Float64Slice(prices))
	return prices
}

// Returns sorted bid prices.
func SortedBidPrices() []float64 {
	prices := make([]float64, 0, len(Bids))
	for k := range Bids {
		prices = append(prices, k)
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(prices)))
	return prices
}

func StartMonitor(productCode string, interval time.Duration) {

	// call rest api regularly
	go getBoardRegularly(productCode, interval)
	go getBoardStateRegularly(productCode, interval)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	client := bitflyergo.WebSocketClient{
		Debug: false,
	}
	client.Connect()
	defer client.Con.Close()

	client.SubscribeBoardSnapshot(productCode)
	client.SubscribeBoard(productCode)
	client.SubscribeExecutions(productCode)

	brdSnpCh := make(chan bitflyergo.Board)
	brdCh := make(chan bitflyergo.Board)
	excCh := make(chan []bitflyergo.Execution)
	go client.Receive(brdSnpCh, brdCh, excCh, nil, nil)

LOOP:
	for {
		select {
		case e := <-excCh: // receive execution history
			Ltp = e[len(e)-1].Price
		case b := <-brdSnpCh: // receive board snapshot
			updateBoard(&b, true)
		case b := <-brdCh: // board difference
			updateBoard(&b, false)
		case _ = <-interrupt:
			client.Con.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			break LOOP
		}
	}
}

// Call 'getboard' api regularly.
func getBoardRegularly(productCode string, interval time.Duration) {
	for {
		board, err := restClient.GetBoard(productCode)
		if err != nil {
			log.Fatal(err)
		}
		BidTotalSize = board.TotalBidSize()
		AskTotalSize = board.TotalAskSize()
		time.Sleep(interval)
	}
}

// Call 'getboardstate' api regularly.
func getBoardStateRegularly(productCode string, interval time.Duration) {
	for {
		boardState, err := restClient.GetBoardState(productCode)
		if err != nil {
			log.Fatal(err)
		}
		Health = boardState.Health
		time.Sleep(interval)
	}
}

// Update board.
//
// if refresh true, dispose current board before update.
func updateBoard(b *bitflyergo.Board, refresh bool) {
	BoardLock.Lock()

	if refresh {
		Bids = map[float64]float64{}
		Asks = map[float64]float64{}
	}

	// update asks
	for price, size := range b.Asks {
		if size > 0 {
			Asks[price] = size
		} else {
			if _, ok := Asks[price]; ok {
				delete(Asks, price)
			}
		}
	}

	// update bids
	for price, size := range b.Bids {
		if size > 0 {
			Bids[price] = size
		} else {
			if _, ok := Bids[price]; ok {
				delete(Bids, price)
			}
		}
	}

	// update mid
	MidPrice = b.MidPrice

	BoardLock.Unlock()
}
