// Package model runs a versioned, synthetic baseline and bounded personal corrections.
package model

import (
	"embed"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

//go:embed baseline.json
var assets embed.FS

var FeatureNames = []string{"bias", "remaining_log", "lead_log", "effort", "active_load", "soon_load", "description_missing", "text_length", "recurring", "importance", "due_weekend", "cat_work", "cat_study", "cat_finance", "cat_home", "cat_health", "cat_personal", "cat_other"}

type Model struct {
	Schema       int      `json:"schema"`
	Version      string   `json:"version"`
	Categories   []string `json:"categories"`
	FeatureNames []string `json:"feature_names"`
	Text         struct {
		Vocabulary []string    `json:"vocabulary"`
		Weights    [][]float64 `json:"log_likelihood"`
		Prior      []float64   `json:"log_prior"`
	} `json:"text"`
	RiskWeights []float64 `json:"risk_weights"`
	positions   map[string]int
}

type Input struct {
	Title, Description                                 string
	DueAt, CreatedAt                                   time.Time
	Importance, EstimatedMinutes, ActiveLoad, SoonLoad int
	Recurring                                          bool
}

type Prediction struct {
	Category           string   `json:"category"`
	CategoryConfidence float64  `json:"category_confidence"`
	Risk               float64  `json:"risk"`
	Score              float64  `json:"score"`
	Level              string   `json:"level"`
	Reasons            []string `json:"reasons"`
	ModelVersion       string   `json:"model_version"`
}

type Example struct {
	Features []float64
	Late     bool
}

func Load() (*Model, error) {
	data, err := assets.ReadFile("baseline.json")
	if err != nil {
		return nil, err
	}
	return Parse(data)
}
func Parse(data []byte) (*Model, error) {
	var m Model
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Schema != 1 || m.Version == "" || len(m.Categories) != 7 || len(m.Text.Prior) != 7 || len(m.Text.Weights) != 7 || len(m.RiskWeights) != len(FeatureNames) || len(m.FeatureNames) != len(FeatureNames) {
		return nil, fmt.Errorf("invalid ML model dimensions")
	}
	for i, name := range FeatureNames {
		if m.FeatureNames[i] != name {
			return nil, fmt.Errorf("incompatible ML features")
		}
	}
	m.positions = map[string]int{}
	for i, w := range m.Text.Vocabulary {
		m.positions[w] = i
	}
	for _, row := range m.Text.Weights {
		if len(row) != len(m.Text.Vocabulary) {
			return nil, fmt.Errorf("invalid classifier dimensions")
		}
		for _, v := range row {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return nil, fmt.Errorf("nonfinite model")
			}
		}
	}
	for _, v := range m.RiskWeights {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("nonfinite model")
		}
	}
	return &m, nil
}

func tokenize(title, description string) map[string]int {
	s := strings.ReplaceAll(strings.ToLower(title+" "+title+" "+description), "ё", "е")
	words := strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
	out := map[string]int{}
	for _, word := range words {
		r := []rune(word)
		if len(r) < 3 {
			continue
		}
		if len(r) > 7 {
			r = r[:7]
		}
		key := string(r)
		if out[key] < 3 {
			out[key]++
		}
	}
	return out
}
func (m *Model) Classify(title, description string) (string, []float64) {
	scores := append([]float64(nil), m.Text.Prior...)
	recognized := false
	for word, n := range tokenize(title, description) {
		if index, ok := m.positions[word]; ok {
			recognized = true
			for c := range scores {
				scores[c] += m.Text.Weights[c][index] * float64(n)
			}
		}
	}
	other := func() (string, []float64) { p := make([]float64, 7); p[6] = 1; return "other", p }
	if !recognized {
		return other()
	}
	maximum := scores[0]
	for _, v := range scores {
		maximum = math.Max(maximum, v)
	}
	total := 0.
	for i := range scores {
		scores[i] = math.Exp(scores[i] - maximum)
		total += scores[i]
	}
	best := 0
	for i := range scores {
		scores[i] /= total
		if scores[i] > scores[best] {
			best = i
		}
	}
	if scores[best] < .52 {
		return other()
	}
	return m.Categories[best], scores
}
func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
func Sigmoid(v float64) float64       { return 1 / (1 + math.Exp(-clamp(v, -30, 30))) }
func (m *Model) Features(in Input, now time.Time) []float64 {
	_, probs := m.Classify(in.Title, in.Description)
	norm := func(hours float64) float64 { return math.Log1p(clamp(hours, 0, 720)) / math.Log(721) }
	effort := in.EstimatedMinutes
	if effort <= 0 {
		effort = 30
	}
	importance := in.Importance
	if importance < 1 || importance > 3 {
		importance = 2
	}
	missing := 0.
	if strings.TrimSpace(in.Description) == "" {
		missing = 1
	}
	recurring := 0.
	if in.Recurring {
		recurring = 1
	}
	weekend := 0.
	if day := in.DueAt.UTC().Weekday(); day == time.Saturday || day == time.Sunday {
		weekend = 1
	}
	x := []float64{1, norm(in.DueAt.Sub(now).Hours()), norm(in.DueAt.Sub(in.CreatedAt).Hours()), clamp(float64(effort), 1, 480) / 480, clamp(float64(in.ActiveLoad), 0, 20) / 20, clamp(float64(in.SoonLoad), 0, 10) / 10, missing, clamp(float64(utf8.RuneCountInString(in.Title)+utf8.RuneCountInString(in.Description)), 0, 2000) / 2000, recurring, float64(importance-1) / 2, weekend}
	return append(x, probs...)
}
func (m *Model) Risk(features, delta []float64, samples int) float64 {
	z := 0.
	strength := clamp(float64(samples)/20, 0, 1)
	for i, w := range m.RiskWeights {
		if i >= len(features) {
			break
		}
		z += w * features[i]
		if len(delta) == len(m.RiskWeights) {
			z += clamp(delta[i], -.75, .75) * strength * features[i]
		}
	}
	return Sigmoid(z)
}
func (m *Model) Predict(in Input, now time.Time, delta []float64, samples int) Prediction {
	category, probs := m.Classify(in.Title, in.Description)
	confidence := 0.
	for _, p := range probs {
		confidence = math.Max(confidence, p)
	}
	risk := m.Risk(m.Features(in, now), delta, samples)
	hours := in.DueAt.Sub(now).Hours()
	urgency := 1 / (1 + math.Max(0, hours)/24)
	importance := in.Importance
	if importance < 1 || importance > 3 {
		importance = 2
	}
	reasons := []string{}
	if hours < 0 {
		risk = 1
		reasons = append(reasons, "Срок уже прошёл")
	} else if hours <= 24 {
		reasons = append(reasons, "Дедлайн в ближайшие сутки")
	}
	if hours >= 0 && risk >= .65 {
		reasons = append(reasons, "Повышенная оценка риска просрочки")
	}
	if importance == 3 {
		reasons = append(reasons, "Вы задали высокую важность")
	}
	if in.SoonLoad >= 3 {
		reasons = append(reasons, "Несколько других сроков в ближайшие сутки")
	}
	if samples >= 5 {
		reasons = append(reasons, "Учтена ваша история выполнения")
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "Есть запас времени до дедлайна")
	}
	score := math.Round((.5*urgency+.3*risk+.2*float64(importance-1)/2)*1000) / 10
	level := "low"
	if score >= 65 {
		level = "high"
	} else if score >= 40 {
		level = "medium"
	}
	return Prediction{Category: category, CategoryConfidence: confidence, Risk: risk, Score: score, Level: level, Reasons: reasons, ModelVersion: m.Version}
}

// FitPersonal refits from current valid outcomes, so edits/reopening/deletion can retract a label.
func (m *Model) FitPersonal(examples []Example) []float64 {
	delta := make([]float64, len(m.RiskWeights))
	if len(examples) == 0 {
		return delta
	}
	for epoch := 0; epoch < 20; epoch++ {
		rate := .08 / (1 + float64(epoch)*.15)
		for _, e := range examples {
			if len(e.Features) != len(delta) {
				continue
			}
			z := 0.
			valid := true
			for i, x := range e.Features {
				if math.IsNaN(x) || math.IsInf(x, 0) || x < 0 || x > 1 {
					valid = false
					break
				}
				z += (m.RiskWeights[i] + delta[i]) * x
			}
			if !valid {
				continue
			}
			y := 0.
			if e.Late {
				y = 1
			}
			error := Sigmoid(z) - y
			for i, x := range e.Features {
				delta[i] = clamp(delta[i]-rate*(error*x+.5*delta[i]), -.75, .75)
			}
		}
	}
	return delta
}
