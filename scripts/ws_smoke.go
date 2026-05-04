package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

type tokenResp struct {
	Token string `json:"token"`
}

type bookingResp struct {
	Booking struct {
		ID string `json:"id"`
	} `json:"booking"`
}

func main() {
	baseURL := env("BASE_URL", "http://localhost:8080")
	wsURL := env("WS_URL", "ws://localhost:8080/ws")
	roomID := env("ROOM_ID", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	slotID := env("SLOT_ID", "99999999-9999-9999-9999-999999999999")

	token, err := login(baseURL, `{"role":"user"}`)
	if err != nil {
		fatalf("dummyLogin: %v", err)
	}

	connA, _, err := websocket.DefaultDialer.Dial(wsURL+"?token="+token, nil)
	if err != nil {
		fatalf("ws dial A: %v", err)
	}
	defer connA.Close()

	connB, _, err := websocket.DefaultDialer.Dial(wsURL+"?token="+token, nil)
	if err != nil {
		fatalf("ws dial B: %v", err)
	}
	defer connB.Close()

	sub := map[string]any{"type": "subscribe", "roomId": roomID}
	if err := subscribe(connA, sub); err != nil {
		fatalf("ws subscribe A: %v", err)
	}
	if err := subscribe(connB, sub); err != nil {
		fatalf("ws subscribe B: %v", err)
	}
	fmt.Println("ws ack: both clients subscribed")

	bookingID, err := createBooking(baseURL, token, slotID)
	if err != nil {
		fatalf("create booking: %v", err)
	}

	_ = connB.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, bookedMsg, err := connB.ReadMessage()
	if err != nil {
		fatalf("ws read slot_booked on client B: %v", err)
	}
	fmt.Printf("event1 (client B): %s\n", string(bookedMsg))

	if err := cancelBooking(baseURL, token, bookingID); err != nil {
		fatalf("cancel booking: %v", err)
	}

	_ = connB.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, releasedMsg, err := connB.ReadMessage()
	if err != nil {
		fatalf("ws read slot_released on client B: %v", err)
	}
	fmt.Printf("event2 (client B): %s\n", string(releasedMsg))
}

func login(baseURL, body string) (string, error) {
	respBody, err := post(baseURL+"/dummyLogin", "", body)
	if err != nil {
		return "", err
	}
	var tr tokenResp
	if err := json.Unmarshal(respBody, &tr); err != nil {
		return "", err
	}
	return tr.Token, nil
}

func createBooking(baseURL, token, slotID string) (string, error) {
	respBody, err := post(baseURL+"/bookings/create", token, fmt.Sprintf(`{"slotId":"%s","createConferenceLink":false}`, slotID))
	if err != nil {
		return "", err
	}
	var br bookingResp
	if err := json.Unmarshal(respBody, &br); err != nil {
		return "", err
	}
	return br.Booking.ID, nil
}

func cancelBooking(baseURL, token, bookingID string) error {
	_, err := post(baseURL+"/bookings/"+bookingID+"/cancel", token, "")
	return err
}

func subscribe(conn *websocket.Conn, msg map[string]any) error {
	if err := conn.WriteJSON(msg); err != nil {
		return err
	}
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, ack, err := conn.ReadMessage()
	if err != nil {
		return err
	}
	fmt.Printf("subscribe ack: %s\n", string(ack))
	return nil
}

func post(url, token, body string) ([]byte, error) {
	var rdr io.Reader
	if body != "" {
		rdr = bytes.NewBufferString(body)
	}
	req, err := http.NewRequest(http.MethodPost, url, rdr)
	if err != nil {
		return nil, err
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("status=%d body=%s", resp.StatusCode, string(data))
	}
	return data, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
