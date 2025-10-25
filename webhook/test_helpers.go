package webhook

import "github.com/stretchr/testify/mock"

// MatchWebhook creates a custom matcher for webhook arguments in mocks
func MatchWebhook(matcher func(Webhook) bool) interface{} {
	return mock.MatchedBy(matcher)
}
