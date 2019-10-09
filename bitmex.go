package algo

import (
	"log"
	"math"

	"github.com/block26/TheAlgoV2/settings"
	"github.com/block26/exchanges/bitmex"
	"github.com/block26/exchanges/bitmex/swagger"
)

var config settings.Config

func Connect(settingsFile string, secret bool, algo Algo, rebalance func(float64, *Algo)) {
	config = loadConfiguration(settingsFile, secret)
	// settings = loadConfiguration("dev/mm/testnet", true)
	log.Println(config)
	fireDB := setupFirebase()

	var orders []*swagger.Order
	var b *bitmex.BitMEX

	if config.TestNet {
		b = bitmex.New(bitmex.HostTestnet, config.APIKey, config.APISecret)
	} else {
		b = bitmex.New(bitmex.HostReal, config.APIKey, config.APISecret)
	}

	subscribeInfos := []bitmex.SubscribeInfo{
		{Op: bitmex.BitmexWSOrder, Param: config.Symbol},
		{Op: bitmex.BitmexWSPosition, Param: config.Symbol},
		{Op: bitmex.BitmexWSQuoteBin1m, Param: config.Symbol},
		{Op: bitmex.BitmexWSWallet},
	}

	err := b.Subscribe(subscribeInfos)
	if err != nil {
		log.Fatal(err)
	}

	b.On(bitmex.BitmexWSWallet, func(wallet []*swagger.Wallet, action string) {
		algo.Asset.BaseBalance = float64(wallet[len(wallet)-1].Amount) * 0.00000001
		log.Println("algo.Asset.BaseBalance", algo.Asset.BaseBalance)
	}).On(bitmex.BitmexWSOrder, func(newOrders []*swagger.Order, action string) {
		orders = bitmex.UpdateLocalOrders(orders, newOrders)
	}).On(bitmex.BitmexWSPosition, func(positions []*swagger.Position, action string) {
		position := positions[0]
		algo.Asset.Quantity = float64(position.CurrentQty)
		if math.Abs(algo.Asset.Quantity) > 0 && position.AvgCostPrice > 0 {
			algo.Asset.AverageCost = position.AvgCostPrice
		} else if position.CurrentQty == 0 {
			algo.Asset.AverageCost = 0
		}
		log.Println("AvgCostPrice", algo.Asset.AverageCost, "Quantity", algo.Asset.Quantity)
	}).On(bitmex.BitmexWSQuoteBin1m, func(bins []*swagger.Quote, action string) {
		for _, bin := range bins {
			log.Println(bin.BidPrice)
			rebalance(bin.BidPrice, &algo)
			algo.BuyOrders.Quantity = mulArr(algo.BuyOrders.Quantity, (algo.Asset.Buying * bin.BidPrice))
			algo.SellOrders.Quantity = mulArr(algo.SellOrders.Quantity, (algo.Asset.Selling * bin.BidPrice))
			log.Println("Total Buy BTC", (algo.Asset.Buying))
			log.Println("Total Buy USD", (algo.Asset.Buying * bin.BidPrice))
			log.Println("Total Sell BTC", (algo.Asset.Selling))
			log.Println("Total Sell USD", (algo.Asset.Selling * bin.BidPrice))
			log.Println("Local order length", len(orders))
			log.Println("New order length", len(algo.BuyOrders.Quantity), len(algo.SellOrders.Quantity))
			// log.Println("Buys", algo.BuyOrders.Quantity)
			// log.Println("Sells", algo.SellOrders.Quantity)
			// log.Println("New order length", len(algo.BuyOrders.Price), len(algo.SellOrders.Price))
			b.PlaceOrdersOnBook(config.Symbol, algo.BuyOrders, algo.SellOrders, orders)
			algo.logState(bin.BidPrice)
			updateAlgo(fireDB, "mm")
		}
	})

	b.StartWS()

	forever := make(chan bool)
	<-forever
}
