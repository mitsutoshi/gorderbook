package gorderbook

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mitsutoshi/bitflyergo"
	"github.com/mitsutoshi/boardmonitor/internal/bitflyer"
	"github.com/nsf/termbox-go"
)

const (
	fgColor       = termbox.ColorWhite
	bgColor       = termbox.ColorDefault
	pauseBgColor  = termbox.ColorRed
	boardPriceX   = 1
	boardSizeX    = boardPriceX + 9
	boardSizeBarX = boardSizeX + 11
	sizeBarText   = "-"
	priceFormat   = "%8.0f"
	sizeFormat    = "%8.2f"
	maxBarLength  = 30
)

var (
	width, height int
)

func DrawContents(group int) {

	width, height = termbox.Size()
	termbox.Clear(fgColor, termbox.ColorDefault)

	//drawGuide()

	// draw board label
	drawText(boardPriceX, 1, fmt.Sprintf("%8s", "Ask"), fgColor, bgColor)
	drawText(boardPriceX, height-1, fmt.Sprintf("%8s", "Bid"), fgColor, bgColor)

	// draw board total size
	drawText(boardSizeX, 1, fmt.Sprintf(sizeFormat, bitflyer.BidTotalSize), fgColor, bgColor)
	drawText(boardSizeX, height-1, fmt.Sprintf(sizeFormat, bitflyer.AskTotalSize), fgColor, bgColor)

	// draw status
	var color termbox.Attribute
	switch bitflyer.Health {
	case bitflyergo.HealthNormal:
		color = termbox.ColorGreen
	default:
		color = termbox.ColorRed
	}
	drawText(boardSizeBarX, height/2, fmt.Sprintf("%-10s", bitflyer.Health), color, bgColor)

	bitflyer.BoardLock.Lock()

	askKeys := bitflyer.SortedAskPrices()
	bidKeys := bitflyer.SortedBidPrices()
	spread := askKeys[0] - bidKeys[0]

	if group > 1 {
		var price float64
		var size float64

		// create ask price and size group
		asksGroup := map[float64]float64{}
		price = getAskGroupPrice(askKeys[0], group)
		size = 0.0
		for _, k := range askKeys {
			if k <= price {
				size += bitflyer.Asks[k]
			} else {
				asksGroup[price] = size
				price = getAskGroupPrice(k, group)
				size = bitflyer.Asks[k]
			}
		}

		// sort ask group price
		askGroupKeys := make([]float64, 0, len(asksGroup))
		for k := range asksGroup {
			askGroupKeys = append(askGroupKeys, k)
		}
		sort.Sort(sort.Float64Slice(askGroupKeys))

		// create bid price and size group
		bidsGroup := map[float64]float64{}
		price = getBidGroupPrice(bidKeys[0], group)
		size = 0.0
		for _, k := range bidKeys {
			if k >= price {
				size += bitflyer.Bids[k]
			} else {
				bidsGroup[price] = size
				price = getBidGroupPrice(k, group)
				size = bitflyer.Bids[k]
			}
		}

		// sort bid group price
		bidGroupKeys := make([]float64, 0, len(bidsGroup))
		for k := range bidsGroup {
			bidGroupKeys = append(bidGroupKeys, k)
		}
		sort.Sort(sort.Reverse(sort.Float64Slice(bidGroupKeys)))

		// display board with grouping
		drawBoard(askGroupKeys, asksGroup, bidGroupKeys, bidsGroup, spread, group)

	} else {

		// display board without grouping
		drawBoard(askKeys, bitflyer.Asks, bidKeys, bitflyer.Bids, spread, 1)
	}

	bitflyer.BoardLock.Unlock()
}

func drawGuide() {
	x := width - 40
	drawText(x, 1, "--------------+-----------------", fgColor, bgColor)
	drawText(x, 2, " Command      + Key             ", fgColor, bgColor)
	drawText(x, 3, "--------------+-----------------", fgColor, bgColor)
	drawText(x, 4, " Pause/Resume | space", fgColor, bgColor)
	drawText(x, 5, " Change group | g <price> enter", fgColor, bgColor)
	drawText(x, 6, " Exit         | ctrl+c", fgColor, bgColor)
}

func getAskGroupPrice(price float64, unit int) float64 {
	mod := int(price) % unit
	if mod > 0 {
		return price + float64(unit-mod)
	}
	return price
}

func getBidGroupPrice(price float64, unit int) float64 {
	mod := int(price) % unit
	if mod > 0 {
		return price - float64(mod)
	}
	return price
}

func drawBoard(sortedAskKeys []float64, asks map[float64]float64, sortedBidKeys []float64, bids map[float64]float64, spread float64, group int) {

	rows := height/2 - 2

	// display mid price
	//drawText(boardPriceX, height/2, fmt.Sprintf(priceFormat, midPrice), fgColor, termbox.ColorDefault)

	// display last price
	drawText(boardPriceX, height/2, fmt.Sprintf(priceFormat, bitflyer.Ltp), termbox.ColorYellow, termbox.ColorDefault)

	// display spread
	drawText(boardSizeX, height/2, fmt.Sprintf(priceFormat, spread), fgColor, termbox.ColorDefault)

	maxSize := 0.0
	total := 0.0
	amount := 0.0
	for i, k := range sortedAskKeys {
		amount += k * asks[k]
		total += asks[k]
		if asks[k] > maxSize {
			maxSize = asks[k]
		}
		if i >= rows {
			break
		}
	}
	askAvgPrice := amount / total

	total = 0.0
	amount = 0.0
	for i, k := range sortedBidKeys {
		amount += k * bids[k]
		total += bids[k]
		if bids[k] > maxSize {
			maxSize = bids[k]
		}
		if i >= rows {
			break
		}
	}
	bidAvgPrice := amount / total

	// calculate BTC size per bar
	sizePerBar := maxSize / maxBarLength

	var textColor termbox.Attribute
	var ltpBgColor termbox.Attribute

	ltpInt := int(bitflyer.Ltp)
	gmin := float64(ltpInt - ltpInt%group + 1)
	gmax := float64(ltpInt - ltpInt%group + group)
	y := height/2 - 1

	for _, k := range sortedAskKeys {

		// Draw ask price
		if k == bitflyer.Ltp || (k >= gmin && k <= gmax) {
			textColor = termbox.ColorBlue
			ltpBgColor = termbox.ColorYellow
		} else {
			textColor = fgColor
			ltpBgColor = termbox.ColorDefault
		}

		if askAvgPrice > (k-float64(group)) && askAvgPrice < k {
			textColor = termbox.ColorRed
		}

		drawText(boardPriceX, y, fmt.Sprintf(priceFormat, k), textColor, ltpBgColor)
		drawText(boardSizeX, y, fmt.Sprintf(sizeFormat, asks[k]), fgColor, bgColor)

		// Draw ask size bar
		barLen := int(asks[k] / sizePerBar)
		if barLen > maxBarLength {
			barLen = maxBarLength
		}
		bar := strings.Repeat(sizeBarText, barLen)
		drawText(boardSizeBarX, y, fmt.Sprintf("%-60s", bar), fgColor, bgColor)

		y -= 1
		if y < 2 {
			break
		}
	}

	ltpInt = int(bitflyer.Ltp)
	gmin = float64(ltpInt - ltpInt%group)
	gmax = float64(ltpInt - ltpInt%group + group - 1)
	y = height/2 + 1

	for _, k := range sortedBidKeys {

		// Draw bid price
		if k == bitflyer.Ltp || (k >= gmin && k <= gmax) {
			textColor = termbox.ColorBlue
			ltpBgColor = termbox.ColorYellow
		} else {
			textColor = fgColor
			ltpBgColor = termbox.ColorDefault
		}

		if bidAvgPrice > (k-float64(group)) && bidAvgPrice < k {
			textColor = termbox.ColorGreen
		}

		drawText(boardPriceX, y, fmt.Sprintf(priceFormat, k), textColor, ltpBgColor)
		drawText(boardSizeX, y, fmt.Sprintf(sizeFormat, bids[k]), fgColor, bgColor)

		// Draw bid size bar
		barLen := int(bids[k] / sizePerBar)
		if barLen > maxBarLength {
			barLen = maxBarLength
		}
		bar := strings.Repeat(sizeBarText, barLen)
		drawText(boardSizeBarX, y, fmt.Sprintf("%-60s", bar), fgColor, bgColor)

		y += 1
		if y > height-2 {
			break
		}
	}
}

func DrawPauseSign() {
	x := width/2 - 6
	y := height / 2
	drawText(x, y-1, fmt.Sprintf("%13s", " "), fgColor, pauseBgColor)
	drawText(x, y, fmt.Sprintf("%s", "    Pause    "), fgColor, pauseBgColor)
	drawText(x, y+1, fmt.Sprintf("%13s", " "), fgColor, pauseBgColor)
}

func drawText(x, y int, text string, fg termbox.Attribute, bg termbox.Attribute) {
	for _, c := range text {
		termbox.SetCell(x, y, c, fg, bg)
		x++
	}
}
