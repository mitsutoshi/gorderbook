package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/mitsutoshi/boardmonitor/internal/bitflyer"
	"github.com/mitsutoshi/boardmonitor/internal/gorderbook"
	"github.com/nsf/termbox-go"
)

const (
	productCode           = "FX_BTC_JPY"
	displayUpdateInterval = 150 * time.Millisecond
	watchInterval         = 5 * time.Second
)

var (
	group            *int
	pause            = false
	lastModifiedTime time.Time
)

func main() {

	// parse options
	group = flag.Int("group", 1, "grouping price on board.")
	flag.Parse()

	// init log file
	f, err := os.OpenFile("gorderbook.log", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(f)

	run()
}

func run() {

	// start to monitor board
	go bitflyer.StartMonitor(productCode, watchInterval)

	if err := termbox.Init(); err != nil {
		log.Fatal(err)
	}
	defer termbox.Close()
	termbox.SetOutputMode(termbox.Output256)
	eventQueue := make(chan termbox.Event)
	go func() {
		for {
			eventQueue <- termbox.PollEvent()
		}
	}()

LOOP:
	for {
		select {

		// received operating of user
		case event := <-eventQueue:
			switch event.Type {
			case termbox.EventKey:
				switch event.Key {
				case termbox.KeyCtrlC: // suspend
					break LOOP
				case termbox.KeySpace: // pause update display
					pause = !pause
				}
			}
		default:
		}

		// update display contents
		if needsUpdateDisplay() {
			if !pause {
				gorderbook.DrawContents(*group)
			} else {
				gorderbook.DrawPauseSign()
			}
			lastModifiedTime = time.Now()
			termbox.Flush()
		}
	}
}

func needsUpdateDisplay() bool {
	return time.Now().Sub(lastModifiedTime) >= displayUpdateInterval &&
		len(bitflyer.Asks) > 0 && len(bitflyer.Bids) > 0
}
