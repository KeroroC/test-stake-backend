package cache

import (
	"testing"
)

func TestBuildListKey_Deterministic(t *testing.T) {
	query := struct {
		Page     int
		PageSize int
		User     string
	}{Page: 1, PageSize: 20, User: "0xabc"}

	key1 := BuildListKey("staked", query)
	key2 := BuildListKey("staked", query)
	if key1 != key2 {
		t.Errorf("BuildListKey not deterministic: %s != %s", key1, key2)
	}
	if key1 == "" {
		t.Error("BuildListKey returned empty string")
	}
}

func TestBuildListKey_DifferentInputs(t *testing.T) {
	q1 := struct {
		Page     int
		PageSize int
	}{Page: 1, PageSize: 20}
	q2 := struct {
		Page     int
		PageSize int
	}{Page: 2, PageSize: 20}

	k1 := BuildListKey("staked", q1)
	k2 := BuildListKey("staked", q2)
	if k1 == k2 {
		t.Errorf("different queries should produce different keys, both got %s", k1)
	}
}

func TestBuildListKey_DifferentEventTypes(t *testing.T) {
	query := struct {
		Page int
	}{Page: 1}

	k1 := BuildListKey("staked", query)
	k2 := BuildListKey("withdrawn", query)
	if k1 == k2 {
		t.Errorf("different event types should produce different keys, both got %s", k1)
	}
}
