package main

// Subscription - basic struct for push subsriptions
type Subscription struct {
	DeviceID  string  `json:"deviceid"`
	Token     string  `json:"token"`
	Type      string  `json:"type"`
	Base      string  `json:"base"`
	Currency  string  `json:"currency"`
	Price     float64 `json:"price"`
	Condition string  `json:"condition"`
	Lang      string  `json:"lang"`
	Date      string  `json:"date"`
}

// ValidType - Check if subsription type is valid
func ValidType(t string) bool {
	switch t {
	case
		"general",
		"price",
		"daly":
		return true
	}
	return false
}

// Put Subscribtion to DB
func putSubscription(s Subscription) {
	if s.Type == "general" {
		s.Base = ""
	}
	if !isSubscriptionInDB(s) {
		writeNewSubscription(s)
		return
	}
}

// Update Token if changed
//func updSub(s Subscription) {
//	fmt.Printf("Here should be some update subscribtion logic\n")
//}
