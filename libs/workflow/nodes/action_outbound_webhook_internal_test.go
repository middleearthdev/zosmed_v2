package nodes

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/zosmed/zosmed/libs/workflow"
)

// fakeDoer is an injectable httpDoer so Execute can be exercised without a
// real network call (and without tripping the SSRF guard on loopback URLs).
type fakeDoer struct {
	status int
	err    error
	gotReq *http.Request
}

func (d *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	d.gotReq = req
	if d.err != nil {
		return nil, d.err
	}
	return &http.Response{StatusCode: d.status, Body: io.NopCloser(strings.NewReader("ok"))}, nil
}

func TestOutboundWebhook_MissingURLRejectedAtBuild(t *testing.T) {
	if _, err := BuildOutboundWebhook(json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected Build error when config.url is empty")
	}
}

func TestOutboundWebhook_SSRFHostsRejectedAtBuild(t *testing.T) {
	for _, raw := range []string{
		`{"url":"http://localhost/hook"}`,
		`{"url":"http://127.0.0.1/hook"}`,
		`{"url":"http://10.0.0.5/hook"}`,
		`{"url":"http://192.168.1.10/hook"}`,
		`{"url":"http://169.254.169.254/latest"}`, // link-local / cloud metadata
		`{"url":"ftp://example.com/hook"}`,        // non-http scheme
	} {
		if _, err := BuildOutboundWebhook(json.RawMessage(raw)); err == nil {
			t.Errorf("expected Build to reject SSRF/unsafe url: %s", raw)
		}
	}
}

func TestOutboundWebhook_PublicLiteralIPBuilds(t *testing.T) {
	// A literal public IP resolves without a real DNS query and passes the guard.
	if _, err := BuildOutboundWebhook(json.RawMessage(`{"url":"http://8.8.8.8/hook"}`)); err != nil {
		t.Fatalf("expected a public literal-IP url to build, got: %v", err)
	}
}

func TestOutboundWebhook_SignatureRequiresSecret(t *testing.T) {
	if _, err := BuildOutboundWebhook(json.RawMessage(`{"url":"http://8.8.8.8/h","includeSignature":true}`)); err == nil {
		t.Fatal("expected Build error when includeSignature is true but secret is empty")
	}
}

func webhookEvent() workflow.Event {
	return workflow.Event{Source: workflow.SourceComment, AccountID: "acc-1", FromUsername: "rina", Text: "hai"}
}

func TestOutboundWebhook_Non2xxDoesNotFailRun(t *testing.T) {
	a := &outboundWebhookAction{url: "http://8.8.8.8/h", client: &fakeDoer{status: 500}}
	res, err := a.Execute(context.Background(), &workflow.RunContext{Event: webhookEvent()})
	if err != nil {
		t.Fatalf("a non-2xx response must not fail the run, got err: %v", err)
	}
	if !strings.Contains(res.Detail, "non-2xx") {
		t.Errorf("detail = %q, want it to mention non-2xx", res.Detail)
	}
}

func TestOutboundWebhook_TransportErrorDoesNotFailRun(t *testing.T) {
	a := &outboundWebhookAction{url: "http://8.8.8.8/h", client: &fakeDoer{err: errors.New("dial timeout")}}
	if _, err := a.Execute(context.Background(), &workflow.RunContext{Event: webhookEvent()}); err != nil {
		t.Fatalf("a transport error must not fail the run, got err: %v", err)
	}
}

func TestOutboundWebhook_SuccessSetsSignatureHeader(t *testing.T) {
	doer := &fakeDoer{status: 200}
	a := &outboundWebhookAction{url: "http://8.8.8.8/h", includeSignature: true, secret: "s3cr3t", client: doer}
	res, err := a.Execute(context.Background(), &workflow.RunContext{Event: webhookEvent()})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !strings.Contains(res.Detail, "200") {
		t.Errorf("detail = %q, want status 200 mentioned", res.Detail)
	}
	if got := doer.gotReq.Header.Get(outboundWebhookSignatureHeader); !strings.HasPrefix(got, "sha256=") {
		t.Errorf("expected %s header with sha256= prefix, got %q", outboundWebhookSignatureHeader, got)
	}
}
