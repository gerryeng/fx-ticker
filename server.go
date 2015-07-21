package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

// rate acts a cache for the calculated rate
var usdSgdRate float64
var sgdUsdRate float64

const (

	// Fees that each exchange charges
	FEES_PERCENT = 2.00

	// How frequently should we update the rates?
	POLL_INTERVAL_SECONDS = 1
)

func CoinbasePrice() (buyPrice float64, sellPrice float64, err error) {
	priceJson, err := httpGetJson("https://api.exchange.coinbase.com/products/BTC-USD/ticker")
	if err != nil {
		return
	}

	price, err := strconv.ParseFloat(priceJson["price"].(string), 64)
	if err != nil {
		return
	}

	buyPrice = price
	sellPrice = price

	return
}

func CoinHakoPrice() (buyPrice float64, sellPrice float64, err error) {
	priceJson, err := httpGetJson("https://coinhako.com/api/v1/price/currency/BTCSGD")
	if err != nil {
		return
	}

	priceData := priceJson["data"].(map[string]interface{})
	buyPrice, err = strconv.ParseFloat(priceData["buy_price"].(string), 64)
	if err != nil {
		return
	}

	sellPrice, err = strconv.ParseFloat(priceData["sell_price"].(string), 64)
	if err != nil {
		return
	}

	return
}

func USDSGDRates() (usdSgdRate float64, sgdUsdRate float64, err error) {
	cbBuyPrice, cbSellPrice, err := CoinbasePrice()
	if err != nil {
		return
	}

	chBuyPrice, chSellPrice, err := CoinHakoPrice()
	if err != nil {
		return
	}

	usdSgdRate = chSellPrice / cbBuyPrice * (1 + FEES_PERCENT/100)
	sgdUsdRate = cbSellPrice / chBuyPrice * (1 + FEES_PERCENT/100)

	return
}

func RatesPoller() {
	for {
		usdSgdResp, sgdUsdResp, err := USDSGDRates()
		if err != nil {
			panic(err)
		}
		usdSgdRate = usdSgdResp
		sgdUsdRate = sgdUsdResp
		time.Sleep(time.Second * POLL_INTERVAL_SECONDS)
	}
}

func handleRateRequest(c *gin.Context) {
	c.JSON(200, gin.H{
		"USDSGD": gin.H{
			"rate":          fmt.Sprintf("%.4f", usdSgdRate),
			"buy_exchange":  "coinbase",
			"sell_exchange": "coinhako",
		},
		"SGDUSD": gin.H{
			"rate":          fmt.Sprintf("%.4f", sgdUsdRate),
			"buy_exchange":  "coinhako",
			"sell_exchange": "coinbase",
		},
	})
}

func StartServer() {

	go RatesPoller()

	r := gin.Default()
	r.Use(corsMiddleware())
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

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	}
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
