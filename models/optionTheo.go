package models

import (
	"fmt"
	"math"
	// "strconv"
	"time"

	"github.com/chobie/go-gaussian"
)

type OptionTheo struct {
	Strike          float64 // Strike price
	UnderlyingPrice float64 // Underlying price
	InterestRate    float64 // Risk free rate (assume 0)
	Volatility      float64 // Implied volatility
	CurrentTime     int     // Current time (ms)
	Expiry          int     // Expiration date (ms)
	TimeLeft        float64 // Time left until expiry (days)
	OptionType      string  // "call" or "put"
	Theo            float64 // Theoretical value calculated via Black-Scholes
	BinomialTheo    float64 // Theoretical value calculated via binomial tree
	Delta           float64 // Change in theo wrt. 1 USD change in UnderlyingPrice
	Theta           float64 // Change in theo wrt. 1 day decrease in timeLeft
	Gamma           float64 // Change in delta wrt. 1 USD change in UnderlyingPrice
	Vega            float64 // Change in theo wrt. 1% increase in volatility
	WeightedVega    float64 // Vega / Vega of ATM option
}

const PI float64 = 3.14159265359
const day = 86400

// Either theo or volatility is unknown (pass in -1.0 for unknown values)
func NewOptionTheo(optionType string, UnderlyingPrice float64, strike float64,
	currentTime int, expiry int, r float64,
	volatility float64, theo float64) *OptionTheo {
	o := &OptionTheo{
		Strike:          strike,
		UnderlyingPrice: UnderlyingPrice,
		InterestRate:    r,
		CurrentTime:     currentTime,
		Expiry:          expiry,
		TimeLeft:        getTimeLeft(currentTime, expiry),
		OptionType:      optionType,
		Volatility:      volatility,
		Theo:            theo,
	}
	return o
}

func (o *OptionTheo) String() string {
	return fmt.Sprintf("%v %v with expiry %v\n", o.Strike, o.OptionType, o.getExpiryString())
}

func (o *OptionTheo) getExpiryString() string {
	return time.Unix(int64(o.Expiry/1000), 0).UTC().String()
}

// Times in ms; return time in days
func getTimeLeft(currentTime int, expiry int) float64 {
	return float64(expiry-currentTime) / float64(1000*day*365)
}

func (o *OptionTheo) calcD1(volatility float64) float64 {
	return (math.Log(o.UnderlyingPrice/o.Strike) + (o.InterestRate+(math.Pow(volatility, 2))/2)*o.TimeLeft) / (volatility * math.Sqrt(o.TimeLeft))
}

func (o *OptionTheo) calcD2(volatility float64) float64 {
	return o.calcD1(volatility) - (volatility * math.Sqrt(o.TimeLeft))
}

// Use Black-Scholes pricing model to calculate theoretical option value
func (o *OptionTheo) CalcBlackScholesTheo(calcGreeks bool) {
	if o.Volatility < 0 && o.Theo < 0 {
		fmt.Printf("[%v] Cannot calc theo with negative theo %v and negative vol %v\n", o.String(), o.Theo, o.Volatility)
		return
	}
	norm := gaussian.NewGaussian(0, 1)
	td1 := o.calcD1(o.Volatility)
	td2 := o.calcD2(o.Volatility)
	if o.Volatility < 0 {
		o.CalcVol()
	} else {
		if o.OptionType == "call" {
			o.Theo = (o.UnderlyingPrice*norm.Cdf(td1) - o.Strike*math.Exp(-o.InterestRate*o.TimeLeft)*norm.Cdf(td2)) / o.UnderlyingPrice
		} else if o.OptionType == "put" {
			o.Theo = (o.Strike*math.Exp(-o.InterestRate*o.TimeLeft)*norm.Cdf(-td2) - o.UnderlyingPrice*norm.Cdf(-td1)) / o.UnderlyingPrice
		}
		fmt.Printf("[%v] Calculated theo %v with vol %v, time %v, d1 %v, d2 %v\n", o.String(), o.Theo, o.Volatility, o.TimeLeft, td1, td2)
	}
	if calcGreeks {
		o.CalcGreeks()
	}
}

func (o *OptionTheo) CalcGreeks() {
	norm := gaussian.NewGaussian(0, 1)
	td1 := o.calcD1(o.Volatility)
	td2 := o.calcD2(o.Volatility)
	nPrime := math.Pow((2*PI), -(1/2)) * math.Exp(math.Pow(-0.5*(td1), 2))
	if o.OptionType == "call" {
		o.Delta = norm.Cdf(td1)
		o.Gamma = (nPrime / (o.UnderlyingPrice * o.Volatility * math.Pow(o.TimeLeft, (1/2))))
		o.Theta = (nPrime)*(-o.UnderlyingPrice*o.Volatility*0.5/math.Sqrt(o.TimeLeft)) - o.InterestRate*o.Strike*math.Exp(-o.InterestRate*o.TimeLeft)*norm.Cdf(td2)
	} else if o.OptionType == "put" {
		o.Delta = norm.Cdf(td1) - 1
		o.Gamma = (nPrime / (o.UnderlyingPrice * o.Volatility * math.Pow(o.TimeLeft, (1/2))))
		o.Theta = (nPrime)*(-o.UnderlyingPrice*o.Volatility*0.5/math.Sqrt(o.TimeLeft)) + o.InterestRate*o.Strike*math.Exp(-o.InterestRate*o.TimeLeft)*norm.Cdf(-td2)
	}
}

func (o *OptionTheo) GetBlackScholesTheo(volatility float64) float64 {
	norm := gaussian.NewGaussian(0, 1)
	td1 := o.calcD1(volatility)
	td2 := o.calcD2(volatility)
	theo := 0.
	if o.OptionType == "call" {
		theo = o.UnderlyingPrice*norm.Cdf(td1) - o.Strike*math.Exp(-o.InterestRate*o.TimeLeft)*norm.Cdf(td2)
	} else if o.OptionType == "put" {
		theo = o.Strike*math.Exp(-o.InterestRate*o.TimeLeft)*norm.Cdf(-td2) - o.UnderlyingPrice*norm.Cdf(-td1)
	}
	// fmt.Printf("got theo %v with vol %v, d1 %v d2 %v\n", theo, volatility, td1, td2)
	return theo / o.UnderlyingPrice
}

// Use newton raphson method to find volatility
func (o *OptionTheo) CalcVol() {
	// fmt.Printf("Calculating vol for %v with theo %v, time left %v, underlying %v", o.String(), o.Theo, o.TimeLeft, o.UnderlyingPrice)
	if o.Theo > 0 {
		norm := gaussian.NewGaussian(0, 1)
		v := math.Sqrt(2*PI/o.TimeLeft) * o.Theo
		fmt.Printf("initial vol: %v\n", v)
		for i := 0; i < 10000; i++ {
			d1 := o.calcD1(v)
			d2 := o.calcD2(v)
			vega := o.UnderlyingPrice * norm.Pdf(d1) * math.Sqrt(o.TimeLeft)
			// fmt.Printf("Underlying %v, pdf %v, time el %v\n", o.UnderlyingPrice, norm.Pdf(d1), math.Sqrt(o.TimeLeft))
			cp := 1.0
			if o.OptionType == "put" {
				cp = -1.0
			}
			theo0 := (cp*o.UnderlyingPrice*norm.Cdf(cp*d1) - cp*o.Strike*math.Exp(-o.InterestRate*o.TimeLeft)*norm.Cdf(cp*d2)) / o.UnderlyingPrice
			v = v - (theo0-o.Theo)/vega
			// fmt.Printf("Next vol: %v with theo %v, d1 %v d2 %v vega %v\n", v, theo0, d1, d2, vega)
			if math.Abs(theo0-o.Theo) < math.Pow(10, -25) {
				fmt.Printf("D1: %v, d2: %v\n", d1, d2)
				break
			}
		}
		fmt.Printf("Calculated vol %v for %v, theo %v\n", v, o.String(), o.Theo)
		o.Volatility = v
	} else {
		fmt.Printf("Can only calc vol with positive theo. Found %v\n", o.Theo)
	}
}

func (o *OptionTheo) CalcVega() {
	// fmt.Printf("O theo for %v: %v at underlying price %v\n", o.String(), o.Theo, o.UnderlyingPrice)
	volChange := .01
	newTheo := o.GetBlackScholesTheo(o.Volatility + volChange)
	// fmt.Printf("newTheo %v, original theo %v with vol %v\n", newTheo, o.Theo, o.Volatility)
	o.CalcBlackScholesTheo(false)
	// fmt.Printf("O theo after calc: %v\n", o.Theo)
	o.Vega = (newTheo - o.Theo) * o.UnderlyingPrice
}

func (o *OptionTheo) CalcWeightedVega() {
	atmOption := NewOptionTheo(
		o.OptionType,
		o.UnderlyingPrice,
		o.UnderlyingPrice,
		o.CurrentTime,
		o.Expiry,
		o.InterestRate,
		o.Volatility, // TODO: should we assume ATM volatility here?
		-1.,
	)
	atmOption.CalcBlackScholesTheo(false)
	atmOption.CalcVega()
	o.CalcBlackScholesTheo(false)
	o.CalcVega()
	o.WeightedVega = o.Vega / atmOption.Vega
	// if o.WeightedVega > .05 {
	// 	fmt.Printf("[%v] Got significant weighted vega %v with vega %v and atm vega %v\n", o.String(), o.WeightedVega, o.Vega, atmOption.Vega)
	// }
}

// Get an option's PNL at expiration
func (o *OptionTheo) GetExpiryValue(currentPrice float64) float64 {
	expiryValue := 0.
	if o.OptionType == "call" {
		expiryValue = (currentPrice - o.Strike) / currentPrice
	} else if o.OptionType == "put" {
		expiryValue = (o.Strike - currentPrice) / currentPrice
	}
	if expiryValue < 0 {
		expiryValue = 0
	}
	return expiryValue
}

// TODO: can be made more efficient by assuming paths can intersect (i.e. up -> down yields same node as down -> up)
// Can be done with binomial tree indexing instead of indexing by path string:
//			4
//		2
//	1		5
//		3
//			6
//
// 	0	1	2
//   timestep

// Stopping conditions for binomial walk
const maxPrice = 20000
const minPrice = 2000
const minProb = .00001

// Recursively calculate the expected values of underlying price
func (o *OptionTheo) binomialWalk(move float64, prob float64, currentPrice float64, currentProb float64, path string,
	evSum *float64, timestepsLeft int, walkCache map[string]*float64) {
	value, ok := walkCache[path]
	if ok {
		// fmt.Printf("Loaded EV %v for path %v\n", *value, path)
		*evSum += *value
		return
	} else if timestepsLeft <= 0 || currentPrice > maxPrice || currentPrice < minPrice || currentProb < minProb {
		ev := 0.
		if o.OptionType == "call" {
			ev = (currentPrice - o.Strike) * currentProb
		} else if o.OptionType == "put" {
			ev = (o.Strike - currentPrice) * currentProb
		}
		if ev < 0 {
			ev = 0
		}
		*evSum += ev
		walkCache[path] = &ev
		// log.Printf("Cached EV %v for path %v\n", ev, path)
		// fmt.Printf("Cached EV %v for path %v\n", ev, path)
		return
	}
	currentPrice = currentPrice * (1 + move)
	currentProb = currentProb * prob
	if move < 0 {
		move *= -1
		prob = 1 - prob
		path += "d"
	} else {
		path += "u"
	}
	// Walk up
	o.binomialWalk(move, prob, currentPrice, currentProb, path, evSum, timestepsLeft-1, walkCache)
	// Walk down
	o.binomialWalk(-move, 1-prob, currentPrice, currentProb, path, evSum, timestepsLeft-1, walkCache)
}

// Calculate the theoretical value of an option based on a binary tree model
// We can calculate the appropriate move for each timestep based on volatility of underlying and time to expiry
// Param prob: probability of an upmove at each timestep
// Param numTimesteps: number of timesteps for the binomial tree traversal
func (o *OptionTheo) CalcBinomialTreeTheo(prob float64, numTimesteps int) {
	timestep := o.TimeLeft / float64(numTimesteps)
	move := o.Volatility * math.Sqrt(timestep)
	// fmt.Printf("Calculating binomial tree theo with numTimesteps %v, move %v, prob %v, volatility %v\n", numTimesteps, move, prob, o.volatility)
	path := ""
	walkCache := make(map[string]*float64) // Stores an ev for a path whose ev is known
	evSum := 0.
	o.binomialWalk(move, prob, o.UnderlyingPrice, 1, path, &evSum, numTimesteps, walkCache)
	// Calculate binomial tree theo quoted in terms of underlying price
	o.BinomialTheo = evSum / o.UnderlyingPrice
	// fmt.Printf("EV sum: %v, binomialTheo: %v, move: %v\n", evSum, o.binomialTheo, move)
}
