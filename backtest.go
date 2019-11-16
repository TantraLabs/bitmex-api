package algo

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/c-bata/goptuna"
	"github.com/c-bata/goptuna/successivehalving"
	"github.com/c-bata/goptuna/tpe"
	"github.com/google/uuid"

	"gonum.org/v1/gonum/stat"

	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/tantralabs/TheAlgoV2/models"
	"golang.org/x/sync/errgroup"
	. "gopkg.in/src-d/go-git.v4/_examples"
)

// var minimumOrderSize = 25
var currentRunUUID time.Time

func Optimize(objective func(goptuna.Trial) (float64, error), episodes int) {
	currentRunUUID = time.Now()
	study, err := goptuna.CreateStudy(
		"optmm",
		goptuna.StudyOptionSampler(tpe.NewSampler()),
		goptuna.StudyOptionSetDirection(goptuna.StudyDirectionMaximize),
		goptuna.StudyOptionPruner(successivehalving.NewOptunaPruner()),
		// goptuna.StudyOptionSetDirection(goptuna.StudyDirectionMinimize),
	)

	if err != nil {
		log.Fatal(err)
	}
	//Multithread
	eg, ctx := errgroup.WithContext(context.Background())
	study.WithContext(ctx)
	for i := 0; i < 12; i++ {
		eg.Go(func() error {
			return study.Optimize(objective, episodes)
		})
	}
	if err := eg.Wait(); err != nil {
		log.Fatal(err)
	}

	// Print the best evaluation value and the parameters.
	v, _ := study.GetBestValue()
	p, _ := study.GetBestParams()
	log.Printf("Best evaluation value=%f", v)
	log.Println(p)
}

func RunBacktest(bars []models.Bar, a Algo, rebalance func(float64, Algo) Algo, setupData func(*[]models.Bar, Algo)) float64 {
	// log.Println("Loading Data... ")
	// fmt.Println(unsafe.Sizeof(bars))
	setupData(&bars, a)
	score := runSingleTest(&bars, a, rebalance)
	log.Println("Score", score)
	// optimize(bars)
	return score
}

func runSingleTest(data *[]models.Bar, algo Algo, rebalance func(float64, Algo) Algo) float64 {
	start := time.Now()
	// starting_algo.Asset.BaseBalance := 0
	timestamp := ""
	idx := 0
	log.Println("Running", len(*data), "bars")
	for _, bar := range *data {
		if timestamp == "" {
			log.Println("Start Timestamp", bar.Timestamp)
			// 	//Set average cost if starting with a quote balance
			if algo.Asset.Quantity > 0 {
				algo.Asset.AverageCost = bar.Close
			}
		}
		timestamp = bar.Timestamp
		if idx > algo.DataLength {
			algo.Index = idx
			algo.Asset.Price = bar.Open
			algo = rebalance(bar.Open, algo)
			//Check which buys filled
			pricesFilled, ordersFilled := getFilledBidOrders(algo.BuyOrders.Price, algo.BuyOrders.Quantity, bar.Low)
			fillCost, fillPercentage := algo.getCostAverage(pricesFilled, ordersFilled)
			algo.UpdateBalance(fillCost, algo.Asset.Buying*fillPercentage)

			//Check which sells filled
			pricesFilled, ordersFilled = getFilledAskOrders(algo.SellOrders.Price, algo.SellOrders.Quantity, bar.High)
			fillCost, fillPercentage = algo.getCostAverage(pricesFilled, ordersFilled)
			algo.UpdateBalance(fillCost, algo.Asset.Selling*-fillPercentage)

			// updateBalanceXBTStrat(bar)
			algo.logState(timestamp)
			if algo.Asset.BaseBalance+(algo.Asset.BaseBalance*algo.Asset.Profit) < 0 {
				break
			}
			// algo.History.Balance[len(algo.History.Balance)-1], == portfolio value
			// portfolioValue := algo.History.Balance[len(algo.History.Balance)-1]
		}
		idx++
	}

	elapsed := time.Since(start)
	log.Println("End Timestamp", timestamp)
	//TODO do this during test instead of after the test
	minProfit, maxProfit, _, maxLeverage, drawdown := MinMaxStats(algo.History)

	fmt.Printf("Balance %0.4f \n Cost %0.4f \n Quantity %0.4f \n Max Leverage %0.4f \n Max Drawdown %0.4f \n Max Profit %0.4f \n Max Position Drawdown %0.4f \n Params: %s",
		algo.History[len(algo.History)-1].Balance,
		algo.History[len(algo.History)-1].AverageCost,
		algo.History[len(algo.History)-1].Quantity,
		maxLeverage,
		drawdown,
		maxProfit,
		minProfit,
		createKeyValuePairs(algo.Params),
	)
	log.Println("Execution Speed", elapsed)
	// score := (algo.History[len(algo.History)-1].Balance) + drawdown*3 //+ (minProfit * maxLeverage) - drawdown // maximize
	// score := (algo.History[len(algo.History)-1].Balance) * math.Abs(1/drawdown) //+ (minProfit * maxLeverage) - drawdown // maximize

	percentReturn := make([]float64, len(algo.History))
	last := 0.0
	for i := range algo.History {
		if i == 0 {
			percentReturn[i] = 0
		} else {
			percentReturn[i] = calculateDifference(algo.History[i].Balance, last)
		}
		last = algo.History[i].Balance
	}

	mean, std := stat.MeanStdDev(percentReturn, nil)
	score := mean / std
	score = score * math.Sqrt(365*24*60)

	algo.Result = map[string]interface{}{
		"balance":             algo.History[len(algo.History)-1].UBalance,
		"max_leverage":        maxLeverage,
		"max_position_profit": maxProfit,
		"max_position_dd":     minProfit,
		"max_dd":              drawdown,
		"params":              algo.Params,
		"score":               score,
	}
	//Very primitive score, how much leverage did I need to achieve this balance

	// algo.HistoryFile, err := os.OpenFile("balance.csv", os.O_RDWR|os.O_CREATE, os.ModePerm)
	// if err != nil {
	// 	panic(err)
	// }
	// defer algo.HistoryFile.Close()

	// err = gocsv.MarshalFile(&algo.History, algo.HistoryFile) // Use this to save the CSV back to the file
	// if err != nil {
	// 	panic(err)
	// }

	LogBacktest(algo)
	// score := ((math.Abs(minProfit) / algo.History[len(algo.History)-1].Balance) + maxLeverage) - algo.History[len(algo.History)-1].Balance // minimize
	return score //algo.History.Balance[len(algo.History.Balance)-1] / (maxLeverage + 1)
}

func (algo *Algo) CurrentProfit(price float64) float64 {
	if algo.Asset.Quantity < 0 {
		return calculateDifference(algo.Asset.AverageCost, price)
	} else {
		return calculateDifference(price, algo.Asset.AverageCost)
	}
}

func (algo *Algo) UpdateBalance(fillCost float64, fillAmount float64) {
	// log.Printf("fillCost %.2f -> fillAmount %.2f\n", fillCost, fillCost*fillAmount)
	if math.Abs(fillAmount) > 0 {
		newQuantity := fillCost * fillAmount
		fee := math.Abs(fillAmount) * algo.Asset.MakerFee
		// log.Printf("fillCost %.2f -> fillAmount %.2f -> Fee %.2f \n", fillCost, fillCost*fillAmount, fee)
		currentCost := (algo.Asset.Quantity * algo.Asset.AverageCost)
		totalQuantity := algo.Asset.Quantity + newQuantity
		newCost := fillCost * newQuantity
		if algo.Futures {
			if (newQuantity >= 0 && algo.Asset.Quantity >= 0) || (newQuantity <= 0 && algo.Asset.Quantity <= 0) {
				//Adding to position
				algo.Asset.AverageCost = (math.Abs(newCost) + math.Abs(currentCost)) / math.Abs(totalQuantity)
			} else if ((newQuantity >= 0 && algo.Asset.Quantity <= 0) || (newQuantity <= 0 && algo.Asset.Quantity >= 0)) && math.Abs(newQuantity) >= math.Abs(algo.Asset.Quantity) {
				algo.Asset.AverageCost = fillCost
			} else {
				var diff float64
				if fillAmount > 0 {
					diff = calculateDifference(algo.Asset.AverageCost, fillCost)
				} else {
					diff = calculateDifference(fillCost, algo.Asset.AverageCost)
				}
				algo.Asset.BaseBalance = algo.Asset.BaseBalance + ((math.Abs(newQuantity) * diff) / fillCost)
			}
			algo.Asset.Quantity = algo.Asset.Quantity + newQuantity
		} else {
			if newQuantity >= 0 {
				//Adding to position
				algo.Asset.AverageCost = (math.Abs(newCost) + math.Abs(currentCost)) / math.Abs(totalQuantity)
			}
			// else {
			// 	var diff float64
			// 	if fillAmount > 0 {
			// 		diff = calculateDifference(algo.Asset.AverageCost, fillCost)
			// 	} else {
			// 		diff = calculateDifference(fillCost, algo.Asset.AverageCost)
			// 	}
			// 	algo.Asset.BaseBalance = algo.Asset.BaseBalance + ((math.Abs(newQuantity)*diff) / fillCost)
			// }
			algo.Asset.BaseBalance = algo.Asset.BaseBalance - newCost
			algo.Asset.Quantity = algo.Asset.Quantity + newQuantity
		}
		algo.Asset.BaseBalance = algo.Asset.BaseBalance - fee
	}
}

func calculateDifference(x float64, y float64) float64 {
	//Get percentage difference between 2 numbers
	if y == 0 {
		y = 1
	}
	return (x - y) / y
}

func (algo *Algo) getCostAverage(pricesFilled []float64, ordersFilled []float64) (float64, float64) {
	// print(len(prices), len(orders), len(timestamp_arr[0]))
	percentageFilled := sumArr(ordersFilled)
	if percentageFilled > 0 {
		normalizer := 1 / percentageFilled
		norm := mulArr(ordersFilled, normalizer)
		costAverage := sumArr(mulArrs(pricesFilled, norm))
		return costAverage, percentageFilled
	}
	return 0.0, 0.0
}

func MinMaxStats(history []models.History) (float64, float64, float64, float64, float64) {
	var maxProfit float64 = history[0].Profit
	var minProfit float64 = history[0].Profit

	var maxLeverage float64 = history[0].Leverage
	var minLeverage float64 = history[0].Leverage

	var drawdown float64 = 0.0
	var highestBalance float64 = 0.0

	for _, row := range history {
		if maxProfit < row.Profit {
			maxProfit = row.Profit
		}
		if minProfit > row.Profit {
			minProfit = row.Profit
		}

		if row.UBalance > highestBalance {
			highestBalance = row.UBalance
		}

		if drawdown > row.UBalance-highestBalance {
			drawdown = row.UBalance - highestBalance
		}

		if maxLeverage < row.Leverage {
			maxLeverage = row.Leverage
		}
		if minLeverage > row.Leverage {
			minLeverage = row.Leverage
		}
	}
	return minProfit, maxProfit, minLeverage, maxLeverage, drawdown
}

func getFilledBidOrders(prices []float64, orders []float64, price float64) ([]float64, []float64) {
	var p []float64
	var o []float64
	for i := range prices {
		if prices[i] > price {
			p = append(p, prices[i])
			o = append(o, orders[i])
		}
	}
	return p, o
}

func getFilledAskOrders(prices []float64, orders []float64, price float64) ([]float64, []float64) {
	var p []float64
	var o []float64
	for i := range prices {
		if prices[i] < price {
			p = append(p, prices[i])
			o = append(o, orders[i])
		}
	}
	return p, o
}

func LogBacktest(algo Algo) {
	influx, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     "http://ec2-54-219-145-3.us-west-1.compute.amazonaws.com:8086",
		Username: "russell",
		Password: "KNW(12nAS921D",
	})
	CheckIfError(err)

	bp, _ := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  "backtests",
		Precision: "us",
	})

	uuid := algo.Name + "-" + uuid.New().String()
	tags := map[string]string{
		"algo_name":   algo.Name,
		"run_id":      currentRunUUID.String(),
		"backtest_id": uuid,
	}

	algo.Result["id"] = uuid

	pt, err := client.NewPoint(
		"result",
		tags,
		algo.Result,
		time.Now(),
	)
	bp.AddPoint(pt)

	err = client.Client.Write(influx, bp)
	CheckIfError(err)
	influx.Close()
}
