package auth

import (
	"context"
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/identity"
)

type enrollmentBrokerFake struct {
	owner                         int64
	completion, retention, cancel int
	fail                          bool
}

func (f *enrollmentBrokerFake) CompleteEnrollment(_ context.Context, owner int64, id, state, code string) (identity.Enrollment, error) {
	f.completion++
	if owner != f.owner || id != "enrollment" || state != "native-state" || code != "native-code" {
		return identity.Enrollment{}, identity.ErrDenied
	}
	if f.fail {
		return identity.Enrollment{}, errors.New("sensitive-native-error")
	}
	return identity.Enrollment{State: "completed", Connection: &identity.Connection{OwnerID: owner, ProviderID: identity.Forgejo}}, nil
}

func (f *enrollmentBrokerFake) Enrollment(_ context.Context, owner int64, id string) (identity.Enrollment, error) {
	f.retention++
	if owner != f.owner || id != "enrollment" {
		return identity.Enrollment{}, identity.ErrDenied
	}
	return identity.Enrollment{State: "completed", Connection: &identity.Connection{OwnerID: owner, ProviderID: identity.Forgejo}}, nil
}

func (f *enrollmentBrokerFake) CancelEnrollment(_ context.Context, owner int64, id string) error {
	f.cancel++
	if owner != f.owner || id != "enrollment" {
		return identity.ErrDenied
	}
	return nil
}

func brokerFixture(owner int64) (*Service, *enrollmentBrokerFake) {
	fake := &enrollmentBrokerFake{owner: owner}
	service := New(&config.Config{ForgejoURL: "https://forgejo.example.test"}, nil)
	service.EnrollmentBroker = fake
	return service, fake
}

func brokerEnrollment() identity.Enrollment {
	q := url.Values{"state": {"native-state"}, "redirect_uri": {"https://forgejo.example.test/-/soda/identity/callback"}}
	return identity.Enrollment{ID: "enrollment", ProviderID: identity.Forgejo, VerificationURL: "https://forgejo.example.test/login/oauth/authorize?" + q.Encode()}
}

func bindBroker(t *testing.T, service *Service, owner int64) {
	t.Helper()
	if err := service.BindNativeBrokerEnrollment(owner, brokerEnrollment()); err != nil {
		t.Fatal(err)
	}
}

func callbackBroker(service *Service, query string) *httptest.ResponseRecorder {
	request := httptest.NewRequest("GET", "/identity/callback?"+query, nil)
	recorder := httptest.NewRecorder()
	service.brokerCallback(recorder, request)
	return recorder
}

func TestBrokerCallbackConsumesNativeActorBindingWithoutBrowserCookie(t *testing.T) {
	service, fake := brokerFixture(7)
	bindBroker(t, service, 7)
	response := callbackBroker(service, "state=native-state&code=native-code")
	if response.Code != 200 || fake.completion != 1 || fake.retention != 1 {
		t.Fatal("native broker completion failed without a Soda browser session")
	}
	if len(response.Result().Cookies()) != 0 || response.Header().Get("Cache-Control") != "no-store" ||
		response.Header().Get("Referrer-Policy") != "no-referrer" || strings.Contains(response.Body.String(), "native-code") {
		t.Fatal("callback leaked state, code or credential material")
	}
	if response := callbackBroker(service, "state=native-state&code=native-code"); response.Code != 403 || fake.completion != 1 {
		t.Fatal("callback replay accepted")
	}
}

func TestBrokerCallbackRejectsWrongExpiredAndMalformedState(t *testing.T) {
	service, fake := brokerFixture(7)
	bindBroker(t, service, 7)
	for _, query := range []string{
		"state=foreign-state&code=native-code",
		"state=native-state&state=native-state&code=native-code",
		"state=native-state&code=native-code&error=denied",
	} {
		if response := callbackBroker(service, query); response.Code != 403 {
			t.Fatalf("invalid callback accepted: %s", query)
		}
	}
	if fake.completion != 0 || len(service.brokerBindings) != 1 {
		t.Fatal("invalid callback consumed native binding")
	}
	binding := service.brokerBindings["native-state"]
	binding.expires = time.Now().Add(-time.Second)
	service.brokerBindings["native-state"] = binding
	if response := callbackBroker(service, "state=native-state&code=native-code"); response.Code != 403 || fake.completion != 0 {
		t.Fatal("expired binding accepted")
	}
}

func TestBrokerBindingUsesActorAndEnrollmentAndCanBeCanceled(t *testing.T) {
	service, fake := brokerFixture(42)
	bindBroker(t, service, 42)
	service.ForgetNativeBrokerEnrollment(7, "enrollment")
	if len(service.brokerBindings) != 1 {
		t.Fatal("different actor removed the pending binding")
	}
	service.ForgetNativeBrokerEnrollment(42, "other-enrollment")
	if len(service.brokerBindings) != 1 {
		t.Fatal("different enrollment removed the pending binding")
	}
	service.ForgetNativeBrokerEnrollment(42, "enrollment")
	if len(service.brokerBindings) != 0 {
		t.Fatal("native cancellation retained its broker state")
	}
	if err := service.BindNativeBrokerEnrollment(0, brokerEnrollment()); err == nil {
		t.Fatal("invalid actor admitted")
	}
	if fake.completion != 0 {
		t.Fatal("binding changed credential custody")
	}
}

func TestBrokerBindingRejectsForeignURLAndEnforcesCapacity(t *testing.T) {
	service, _ := brokerFixture(7)
	service.brokerBindings = make(map[string]brokerBinding)
	for _, raw := range []string{
		"https://foreign.example.test/login/oauth/authorize?state=native-state&redirect_uri=https://forgejo.example.test/-/soda/identity/callback",
		"https://forgejo.example.test/login/oauth/authorize?state=native-state&redirect_uri=https://foreign.example.test/callback",
		"https://forgejo.example.test/login/oauth/authorize?state=native-state&state=other&redirect_uri=https://forgejo.example.test/-/soda/identity/callback",
	} {
		enrollment := brokerEnrollment()
		enrollment.VerificationURL = raw
		if err := service.BindNativeBrokerEnrollment(7, enrollment); err == nil {
			t.Fatal("foreign authorization URL accepted")
		}
	}
	for i := 0; i < 128; i++ {
		state := "pending-" + string(rune(i+1))
		service.brokerBindings[state] = brokerBinding{state: state, expires: time.Now().Add(time.Minute)}
	}
	if err := service.BindNativeBrokerEnrollment(7, brokerEnrollment()); err == nil {
		t.Fatal("binding capacity exceeded")
	}
}

func TestBrokerCallbackDoesNotRevealFailureDetails(t *testing.T) {
	service, fake := brokerFixture(7)
	bindBroker(t, service, 7)
	fake.fail = true
	response := callbackBroker(service, "state=native-state&code=native-code")
	if response.Code != 403 || fake.cancel != 1 || strings.Contains(response.Body.String(), "sensitive-native-error") {
		t.Fatal("provider failure detail leaked or enrollment remained live")
	}
}
