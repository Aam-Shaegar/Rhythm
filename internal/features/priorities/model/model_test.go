package model

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func TestPythonGoClassifierParity(t *testing.T) {
	m, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	data, _ := assets.ReadFile("baseline.json")
	var a struct {
		Probes []struct {
			Title, Description, Category string
			Probabilities                []float64
		}
	}
	if err = json.Unmarshal(data, &a); err != nil {
		t.Fatal(err)
	}
	for _, p := range a.Probes {
		category, probs := m.Classify(p.Title, p.Description)
		if category != p.Category {
			t.Fatalf("%s: %s != %s", p.Title, category, p.Category)
		}
		for i, v := range probs {
			if math.Abs(v-p.Probabilities[i]) > 1e-8 {
				t.Fatal("Python/Go classifier mismatch")
			}
		}
	}
}
func TestPriorityAndPersonalization(t *testing.T) {
	m, _ := Load()
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	in := Input{Title: "Купить продукты", Description: "молоко магазин", CreatedAt: now.Add(-time.Hour), DueAt: now.Add(72 * time.Hour), Importance: 2, EstimatedMinutes: 30}
	far := m.Predict(in, now, nil, 0)
	in.DueAt = now.Add(time.Hour)
	soon := m.Predict(in, now, nil, 0)
	if soon.Score <= far.Score || soon.Level != "high" {
		t.Fatal("urgency must raise priority", far, soon)
	}
	in.DueAt = now.Add(-time.Hour)
	if p := m.Predict(in, now, nil, 0); p.Risk != 1 || p.Level != "high" {
		t.Fatal("known overdue outcome", p)
	}
	in.DueAt = now.Add(48 * time.Hour)
	x := m.Features(in, now)
	examples := []Example{}
	for i := 0; i < 40; i++ {
		examples = append(examples, Example{Features: x, Late: true})
	}
	delta := m.FitPersonal(examples)
	if m.Risk(x, delta, 40) <= m.Risk(x, nil, 0) {
		t.Fatal("personal history did not affect risk")
	}
	for _, v := range delta {
		if math.Abs(v) > .75 {
			t.Fatal("unbounded personal weights")
		}
	}
	if m.Risk(x, delta, 0) != m.Risk(x, nil, 0) {
		t.Fatal("no history must use baseline")
	}
}
func TestFeatureNormalizationAndDamagedModel(t *testing.T) {
	m, _ := Load()
	in := Input{DueAt: time.Now().Add(-time.Hour), EstimatedMinutes: 999999, ActiveLoad: 999, Importance: 999}
	for _, v := range m.Features(in, time.Now()) {
		if v < 0 || v > 1 || math.IsNaN(v) {
			t.Fatal(v)
		}
	}
	if _, err := Parse([]byte(`{"schema":1}`)); err == nil {
		t.Fatal("invalid model accepted")
	}
}
