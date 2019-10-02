package interfaces

import (
	"github.com/block26/exchanges/models"
)

type Algo interface {
	rebalance(float64, float64, float64) (models.OrderArray, models.OrderArray)
	connect(settingsFile string, secret bool)
	updateBalance(fillCost float64, fillAmount float64)
	currentProfit(price float64) float64
	runBacktest()
}
