package user

type User struct {
	ID                 int64   `json:"id"`
	TelegramUsername   string  `json:"username"`
	SubscriptionStatus string  `json:"subscription_status"`
	Traffic            float64 `json:"traffic"` // in Mb
}
