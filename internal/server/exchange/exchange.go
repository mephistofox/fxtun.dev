package exchange

const defaultRate = 80.0

var rate float64

// Init sets the USD to RUB exchange rate from config.
func Init(configRate float64) {
	if configRate > 0 {
		rate = configRate
	} else {
		rate = defaultRate
	}
}

// GetRate returns the current USD to RUB exchange rate and a zero timestamp.
func GetRate() (float64, int64) {
	if rate <= 0 {
		return defaultRate, 0
	}
	return rate, 0
}

// ConvertUSDToRUB converts USD amount to RUB with nice rounding (to nearest 5).
func ConvertUSDToRUB(usd float64) float64 {
	r, _ := GetRate()
	return roundToNearest5(usd * r)
}

func roundToNearest5(n float64) float64 {
	return float64(int((n+2.5)/5) * 5)
}
