package scheduler

import (
	"context"
	"log"
	"time"
)

func (s *Scheduler) checkAndUpdateSubscriptions() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	usernames, err := s.db.AllUsername(ctx)
	if err != nil {
		log.Printf("Failed to fetch usernames: %v", err)
		return
	}

	for _, username := range usernames {
		user, err := s.db.User(ctx, username)
		if err != nil {
			log.Printf("Failed to get user %s: %v", username, err)
		}

		if user.Subscription.SubscriptionStatus == "inactive" && user.Subscription.EndSubscription.After(time.Now()) {
			user.Subscription.SubscriptionStatus = "active"
			if err := s.db.UpdateUserSubscription(ctx, username, user.Subscription); err != nil {
				log.Printf("Failed to update subscription for user %s: %v", user.Username, err)
			}
		}

		if user.Subscription.SubscriptionStatus == "active" && user.Subscription.EndSubscription.Before(time.Now()) {
			log.Printf("Subscription expired for user %s, updating status to inactive.", user.Username)
			user.Subscription.SubscriptionStatus = "inactive"
			user.Subscription.EndSubscription = time.Time{}
			if err := s.db.UpdateUserSubscription(ctx, username, user.Subscription); err != nil {
				log.Printf("Failed to update subscription for user %s: %v", user.Username, err)
			}
		}

	}
}
