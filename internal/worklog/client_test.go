package worklog

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestLoginUsesRSAPlaintextAndPersistsCookie(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicKey := base64.StdEncoding.EncodeToString(der)
	client, err := NewClient(Config{BaseURL: "http://example.test", Timeout: time.Second, VerifyTLS: true})
	if err != nil {
		t.Fatal(err)
	}
	client.HTTP.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		headers := make(http.Header)
		headers.Set("Content-Type", "application/json")
		body := ""
		status := http.StatusOK
		switch request.URL.Path {
		case "/rsa/weaver.rsa.GetRsaInfo":
			body = jsonText(map[string]any{"rsa_pub": publicKey, "rsa_code": "-code", "rsa_flag": "FLAG"})
		case "/api/hrm/login/checkLogin":
			_ = request.ParseForm()
			for _, key := range []string{"loginid", "userpassword"} {
				encoded := strings.TrimSuffix(request.Form.Get(key), "FLAG")
				ciphertext, decodeErr := base64.StdEncoding.DecodeString(encoded)
				if decodeErr != nil {
					t.Fatalf("decode %s: %v", key, decodeErr)
				}
				plaintext, decryptErr := rsa.DecryptPKCS1v15(rand.Reader, privateKey, ciphertext)
				if decryptErr != nil {
					t.Fatalf("decrypt %s: %v", key, decryptErr)
				}
				if !strings.HasSuffix(string(plaintext), "-code") {
					t.Fatalf("missing rsa code in %s: %q", key, plaintext)
				}
			}
			headers.Set("Set-Cookie", "ecology_JSessionid=s123; Path=/")
			body = jsonText(map[string]any{"loginstatus": "true"})
		case "/api/hrm/login/remindLogin":
			body = jsonText(map[string]any{})
		case "/api/hrm/login/getAccountList":
			if !strings.Contains(request.Header.Get("Cookie"), "ecology_JSessionid=s123") {
				t.Errorf("cookie was not propagated: %q", request.Header.Get("Cookie"))
			}
			body = jsonText(map[string]any{"status": "1", "data": map[string]any{"userid": "7", "username": "alice", "deptid": "9", "deptname": "研发"}})
		default:
			status = http.StatusNotFound
		}
		return &http.Response{StatusCode: status, Status: http.StatusText(status), Header: headers, Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
	})
	result, err := client.Login(context.Background(), "alice", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if result["userid"] != "7" || client.JSessionID() != "s123" {
		t.Fatalf("unexpected login result: %#v, cookie=%q", result, client.JSessionID())
	}
}

func TestListHistoryParsesHTMLAndHonorsLimit(t *testing.T) {
	client, err := NewClient(Config{BaseURL: "http://example.test", Timeout: time.Second, VerifyTLS: true})
	if err != nil {
		t.Fatal(err)
	}
	client.HTTP.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/api/portal/element/workflowtab" {
			return responseFor(request, http.StatusNotFound, "", nil), nil
		}
		return responseFor(request, http.StatusOK, `{"data":[{"requestname":{"requestid":"1","name":"<span>周报</span>（开始日期:2026-08-03, 结束日期:2026-08-09）","link":"/r/1"},"createtime":"10:00"},{"requestname":{"requestid":"2","name":"第二条"}}]}`, nil), nil
	})
	rows, err := client.ListHistory(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Title != "周报（开始日期:2026-08-03, 结束日期:2026-08-09）" || rows[0].StartDate != "2026-08-03" {
		t.Fatalf("unexpected history rows: %#v", rows)
	}
	if _, err := client.DoForm(context.Background(), "POST", "/api/portal/element/workflowtab", url.Values{}); err != nil {
		t.Fatal(err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

func responseFor(request *http.Request, status int, body string, headers http.Header) *http.Response {
	if headers == nil {
		headers = make(http.Header)
	}
	headers.Set("Content-Type", "application/json")
	return &http.Response{StatusCode: status, Status: http.StatusText(status), Header: headers, Body: io.NopCloser(strings.NewReader(body)), Request: request}
}

func jsonText(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}
