package user

type User struct {
	ID                 int    `json:"id"`
	TelegramUsername   string `json:"telegram_username"`
	SubscriptionStatus string `json:"subscription_status"`
}
