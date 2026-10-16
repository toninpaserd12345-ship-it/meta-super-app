package usecase_test

import (
	"testing"

	"github.com/meta-super-app/backend/internal/usecase"
)

func TestClassifyAdDecision_InsufficientData(t *testing.T) {
	code, badge, _, _, _ := usecase.ClassifyAdDecision(
		0, 0, 50000, 85000, nil, 0, 0, 0, 200000,
	)
	if code != "INSUFFICIENT_DATA" || badge != "secondary" {
		t.Fatalf("expected INSUFFICIENT_DATA, got %s (badge %s)", code, badge)
	}
}

func TestClassifyAdDecision_ScaleWhenProfitableAndVolumeHigh(t *testing.T) {
	realCPA := 35000.0 // target is 50,000
	code, badge, _, _, budget := usecase.ClassifyAdDecision(
		500000, 15000, 50000, 85000, &realCPA, 14, 0, 0, 200000,
	)
	if code != "SCALE" || badge != "success" {
		t.Fatalf("expected SCALE, got %s (badge %s)", code, badge)
	}
	if budget == nil || *budget == "" {
		t.Fatalf("expected proposed budget change, got nil")
	}
}

func TestClassifyAdDecision_HoldWhenCODPendingInTransit(t *testing.T) {
	// Even if CPA is high or unknown, 5 COD orders are in transit (shipping)
	realCPA := 70000.0
	code, badge, reason, _, _ := usecase.ClassifyAdDecision(
		350000, 8000, 50000, 85000, &realCPA, 5, 5, 750000, 200000,
	)
	// COD in transit must prevent premature pausing!
	if code != "HOLD" || badge != "info" {
		t.Fatalf("expected HOLD due to COD in transit, got %s (badge %s)", code, badge)
	}
	if reason == "" {
		t.Fatalf("expected reason explaining COD in transit")
	}
}

func TestClassifyAdDecision_HoldWhenEarlyVolume(t *testing.T) {
	realCPA := 40000.0 // <= target 50,000, but only 4 orders
	code, badge, _, _, _ := usecase.ClassifyAdDecision(
		160000, 5000, 50000, 85000, &realCPA, 4, 0, 0, 200000,
	)
	if code != "HOLD" || badge != "info" {
		t.Fatalf("expected HOLD for early volume (<10 orders), got %s", code)
	}
}

func TestClassifyAdDecision_ImproveWhenAboveTargetBelowBreakEven(t *testing.T) {
	realCPA := 65000.0 // between target 50,000 and break-even 85,000
	code, badge, _, _, _ := usecase.ClassifyAdDecision(
		650000, 20000, 50000, 85000, &realCPA, 10, 0, 0, 200000,
	)
	if code != "IMPROVE" || badge != "warning" {
		t.Fatalf("expected IMPROVE, got %s (badge %s)", code, badge)
	}
}

func TestClassifyAdDecision_PauseWhenCPAExceedsBreakEven(t *testing.T) {
	realCPA := 95000.0 // > break-even 85,000
	code, badge, _, _, _ := usecase.ClassifyAdDecision(
		950000, 25000, 50000, 85000, &realCPA, 10, 0, 0, 200000,
	)
	if code != "PAUSE" || badge != "error" {
		t.Fatalf("expected PAUSE for loss-making CPA, got %s (badge %s)", code, badge)
	}
}

func TestClassifyAdDecision_PauseTestWhenSpendHitsTestBudgetZeroOrders(t *testing.T) {
	// Spend 200,000 Kip, 0 orders
	code, badge, _, _, _ := usecase.ClassifyAdDecision(
		210000, 4500, 50000, 85000, nil, 0, 0, 0, 200000,
	)
	if code != "PAUSE_TEST" || badge != "error" {
		t.Fatalf("expected PAUSE_TEST when spending test budget with 0 orders, got %s", code)
	}
}
