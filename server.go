package main

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

// rate acts a cache for the calculated rate
var rate float64

const (

	// Fees that each exchange charges
	COINBASE_FEES_PERCENT = 1.00
	COINHAKO_FEES_PERCENT = 1.00

	// How frequently should we update the rates?
	POLL_INTERVAL_SECONDS = 1
)

func CoinbaseBuyPrice() (price float64, err error) {
	priceJson, err := httpGetJson("https://api.exchange.coinbase.com/products/BTC-USD/ticker")
	if err != nil {
		return
	}

	price, err = strconv.ParseFloat(priceJson["price"].(string), 64)
	if err != nil {
		return
	}

	return
}

func CoinHakoSellPrice() (price float64, err error) {
	priceJson, err := httpGetJson("https://coinhako.com/api/v1/price/currency/BTCSGD")
	if err != nil {
		return
	}

	priceData := priceJson["data"].(map[string]interface{})
	price, err = strconv.ParseFloat(priceData["buy_price"].(string), 64)
	if err != nil {
		return
	}

	return
}

func USDSGDRate() (rate float64, err error) {
	cbBuyPrice, err := CoinbaseBuyPrice()
	if err != nil {
		return
	}

	// Factor in Coinbase fees
	cbBuyPrice *= (1 - COINBASE_FEES_PERCENT/100)

	chSellPrice, err := CoinHakoSellPrice()
	if err != nil {
		return
	}

	// Factor in Coinhako Fees
	chSellPrice *= (1 + COINHAKO_FEES_PERCENT/100)

	rate = chSellPrice / cbBuyPrice
	return
}

func RatesPoller() {
	for {
		rateResp, err := USDSGDRate()
		if err != nil {
			panic(err)
		}
		rate = rateResp
		time.Sleep(time.Second * POLL_INTERVAL_SECONDS)
	}
}

func handleRateRequest(c *gin.Context) {
	c.JSON(200, gin.H{
		"rate": rate,
	})
}

func StartServer() {

	go RatesPoller()

	r := gin.Default()
	r.GET("/rate", handleRateRequest)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r.Run(":" + port)
}

func main() {
	StartServer()
}

func httpGetJson(url string) (jsonResp map[string]interface{}, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}

	respBlob, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	resp.Body.Close()

	json.Unmarshal(respBlob, &jsonResp)

	return
}
