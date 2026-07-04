package auth

import (
	"sync"
	"testing"

	"golang.org/x/oauth2"
)

func TestConcurrentTokenDeliveryCompletesLoginOnce(t *testing.T) {
	service := NewAuthService("http://127.0.0.1:8287/callback")
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			service.sendToken(&oauth2.Token{AccessToken: "token"})
		}()
	}
	wg.Wait()
	<-service.GetTokenChannel()
	service.sendToken(&oauth2.Token{AccessToken: "late-token"})
	select {
	case <-service.GetTokenChannel():
		t.Fatal("a late callback queued another token after login completed")
	default:
	}
}
