package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Write / Read (base64 cookie)
// ---------------------------------------------------------------------------

func TestCookieWrite_RoundTrip(t *testing.T) {
	rr := httptest.NewRecorder()
	err := Write(rr, http.Cookie{Name: "test", Value: "hello world"})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Copy the Set-Cookie header into a new request so Read can access it.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range rr.Result().Cookies() {
		req.AddCookie(c)
	}

	got, err := Read(req, "test")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestCookieWrite_ValueTooLong(t *testing.T) {
	// A value that, after base64 encoding and cookie overhead, exceeds 4096 bytes.
	long := strings.Repeat("x", 4000)
	rr := httptest.NewRecorder()
	err := Write(rr, http.Cookie{Name: "big", Value: long})
	if err != ErrValueTooLong {
		t.Errorf("Write(long value) = %v, want ErrValueTooLong", err)
	}
}

func TestCookieRead_MissingCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	_, err := Read(req, "absent")
	if err == nil {
		t.Error("Read(absent cookie) expected error, got nil")
	}
}

func TestCookieRead_InvalidBase64(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Set a cookie whose value is not valid base64.
	req.AddCookie(&http.Cookie{Name: "bad", Value: "!!!not-base64!!!"})

	_, err := Read(req, "bad")
	if err != ErrInvalidValue {
		t.Errorf("Read(invalid base64) = %v, want ErrInvalidValue", err)
	}
}

// ---------------------------------------------------------------------------
// WriteSigned / ReadSigned (HMAC-signed cookie)
// ---------------------------------------------------------------------------

func TestSignedCookie_RoundTrip(t *testing.T) {
	key := []byte("supersecretkey32byteslong!!!!!!!!")
	rr := httptest.NewRecorder()
	err := WriteSigned(rr, http.Cookie{Name: "signed", Value: "payload"}, key)
	if err != nil {
		t.Fatalf("WriteSigned: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range rr.Result().Cookies() {
		req.AddCookie(c)
	}

	got, err := ReadSigned(req, key, "signed")
	if err != nil {
		t.Fatalf("ReadSigned: %v", err)
	}
	if got != "payload" {
		t.Errorf("got %q, want %q", got, "payload")
	}
}

func TestSignedCookie_WrongKey(t *testing.T) {
	key := []byte("correct-key-32-bytes-long!!!!!!!!!")
	otherKey := []byte("wrong---key-32-bytes-long!!!!!!!!!")

	rr := httptest.NewRecorder()
	WriteSigned(rr, http.Cookie{Name: "signed", Value: "payload"}, key)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range rr.Result().Cookies() {
		req.AddCookie(c)
	}

	_, err := ReadSigned(req, otherKey, "signed")
	if err != ErrInvalidValue {
		t.Errorf("ReadSigned(wrong key) = %v, want ErrInvalidValue", err)
	}
}

func TestSignedCookie_TamperedValue(t *testing.T) {
	key := []byte("correct-key-32-bytes-long!!!!!!!!!")

	rr := httptest.NewRecorder()
	WriteSigned(rr, http.Cookie{Name: "signed", Value: "original"}, key)

	// Extract the encoded cookie value and mangle the payload portion.
	cookies := rr.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("no cookies set")
	}
	// Replace cookie value with a tampered string that keeps the right length.
	tampered := cookies[0].Value + "X"

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "signed", Value: tampered})

	_, err := ReadSigned(req, key, "signed")
	if err == nil {
		t.Error("ReadSigned(tampered value) expected error, got nil")
	}
}

func TestSignedCookie_TooShort(t *testing.T) {
	key := []byte("key")
	// Manually write a cookie whose decoded value is shorter than sha256.Size.
	rr := httptest.NewRecorder()
	Write(rr, http.Cookie{Name: "short", Value: "tiny"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range rr.Result().Cookies() {
		req.AddCookie(c)
	}

	_, err := ReadSigned(req, key, "short")
	if err != ErrInvalidValue {
		t.Errorf("ReadSigned(too short) = %v, want ErrInvalidValue", err)
	}
}
