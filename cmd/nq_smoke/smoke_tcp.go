package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	ntpkg "nofx/provider/ninjatrader"
)

// runTCPSmoke exercises the Plan 1.5 TCP bridge end-to-end against the
// in-process MockTCPClient. Needs no network, no NT8, no env vars.
// Mirrors the smoke_roundtrip.go shape for the CSV path.
func runTCPSmoke() {
	// Use an ephemeral port so smoke runs don't collide with a live server.
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Printf("FAIL tcp: pick port: %v\n", err)
		os.Exit(1)
	}
	addr := probe.Addr().String()
	_ = probe.Close()

	server := ntpkg.NewTCPServer(nil)
	server.SetAddrForTest(addr)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := server.Start(ctx); err != nil {
		fmt.Printf("FAIL tcp: server start: %v\n", err)
		os.Exit(1)
	}
	defer server.Stop()

	client := ntpkg.NewMockTCPClient(addr, 200*time.Millisecond)
	if err := client.Start(ctx); err != nil {
		fmt.Printf("FAIL tcp: mock client: %v\n", err)
		os.Exit(1)
	}
	defer client.Stop()

	// Wait for the server to register the connection.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if server.IsConnected() {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !server.IsConnected() {
		fmt.Println("FAIL tcp: client did not connect within 2s")
		os.Exit(1)
	}

	start := time.Now()
	sig := ntpkg.SignalPayload{
		Symbol:     "MNQ",
		Side:       "long",
		Quantity:   1,
		Entry:      21500.00,
		StopLoss:   21485.00,
		TakeProfit: 21525.00,
		SignalID:   "smoke-uuid-1",
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
	if err := server.SendSignal(sig); err != nil {
		fmt.Printf("FAIL tcp: send: %v\n", err)
		os.Exit(1)
	}

	select {
	case fill := <-server.Fills():
		elapsed := time.Since(start)
		fmt.Printf("OK tcp: signal->fill in %v (signal_id=%s side=%s price=%.2f status=%s)\n",
			elapsed, fill.SignalID, fill.Side, fill.FillPrice, fill.Status)
	case <-time.After(2 * time.Second):
		fmt.Println("FAIL tcp: no fill within 2s")
		os.Exit(1)
	}
}
