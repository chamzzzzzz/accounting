package normalizer

import (
	"context"
	"sort"
	"strings"
	"unicode"

	"github.com/chamzzzzzz/accounting/sourcedocument"
)

type Spec struct {
	KeyMaxRunes       int
	SameLineYFactor   float64
	RightDistanceMult float64
	BelowDistanceMult float64
	AlignXTolerance   float64
	PairAcceptScore   float64
	KeyWords          []string
	KeySuffixes       []string
}

type Normalizer struct {
	Spec *Spec
}

type unit struct {
	index    int
	text     string
	location *sourcedocument.Location
}

type pairCandidate struct {
	keyIdx   int
	valueIdx int
	score    float64
}

type resultEntry struct {
	ann     *sourcedocument.Annotation
	x       int
	y       int
	valueVu *unit
}

const (
	weightHorizontal = 0.35
	weightVertical   = 0.25
	weightSemantic   = 0.20
	weightPattern    = 0.20
)

type thresholds struct {
	sameLineY int
	rightDist int
	belowDist int
	alignX    int
}

func DefaultSpec() *Spec {
	return &Spec{
		KeyMaxRunes:       14,
		SameLineYFactor:   0.6,
		RightDistanceMult: 8.0,
		BelowDistanceMult: 2.5,
		AlignXTolerance:   0.5,
		PairAcceptScore:   0.55,
		KeyWords:          []string{},
		KeySuffixes:       []string{},
	}
}

func (n *Normalizer) Normalize(_ context.Context, source *sourcedocument.SourceDocument) (*sourcedocument.SourceDocument, error) {
	if source == nil {
		return nil, nil
	}

	spec := resolveSpec(n)
	located, preserved := splitAnnotations(source.Annotations)

	if len(located) == 0 {
		source.Annotations = preserved
		return source, nil
	}

	sortUnitsByPosition(located)
	limits := calculateThresholds(located, spec)

	consumed := make(map[int]bool)
	entries := make([]*resultEntry, 0, len(located)+len(preserved))

	for idx, u := range located {
		ok, key, value := tryDirectColonSplit(u.text)
		if !ok {
			continue
		}
		if !isValidDirectKV(spec, key, value) {
			continue
		}
		consumed[idx] = true
		entries = append(entries, &resultEntry{
			ann:     &sourcedocument.Annotation{Label: normalizeKey(key), Text: strings.TrimSpace(value)},
			x:       u.location.X,
			y:       u.location.Y,
			valueVu: u,
		})
	}

	keyIndices := make([]int, 0, len(located))
	for idx, u := range located {
		if consumed[idx] {
			continue
		}
		if isKeyCandidate(spec, u.text) {
			keyIndices = append(keyIndices, idx)
		}
	}

	pairs := make([]*pairCandidate, 0)
	for _, keyIdx := range keyIndices {
		ku := located[keyIdx]
		for valueIdx, vu := range located {
			if valueIdx == keyIdx || consumed[valueIdx] {
				continue
			}

			hScore, ok := horizontalScore(ku, vu, limits.sameLineY, limits.rightDist)
			vScore, vok := belowScore(ku, vu, limits.belowDist, limits.alignX)
			if !ok && !vok {
				continue
			}

			sem := semanticScore(ku.text, vu.text)
			pat := patternScore(ku.text, vu.text)

			if !ok {
				hScore = 0
			}
			if !vok {
				vScore = 0
			}

			score := weightHorizontal*hScore + weightVertical*vScore + weightSemantic*sem + weightPattern*pat
			if score >= spec.PairAcceptScore {
				pairs = append(pairs, &pairCandidate{keyIdx: keyIdx, valueIdx: valueIdx, score: score})
			}
		}
	}

	sort.SliceStable(pairs, func(i, j int) bool {
		if pairs[i].score != pairs[j].score {
			return pairs[i].score > pairs[j].score
		}
		return pairs[i].keyIdx < pairs[j].keyIdx
	})

	usedKeys := map[int]bool{}
	usedValues := map[int]bool{}
	for _, pair := range pairs {
		if usedKeys[pair.keyIdx] || usedValues[pair.valueIdx] {
			continue
		}
		if consumed[pair.keyIdx] || consumed[pair.valueIdx] {
			continue
		}
		usedKeys[pair.keyIdx] = true
		usedValues[pair.valueIdx] = true
		consumed[pair.keyIdx] = true
		consumed[pair.valueIdx] = true

		ku := located[pair.keyIdx]
		vu := located[pair.valueIdx]
		entries = append(entries, &resultEntry{
			ann:     &sourcedocument.Annotation{Label: normalizeKey(ku.text), Text: strings.TrimSpace(vu.text)},
			x:       ku.location.X,
			y:       ku.location.Y,
			valueVu: vu,
		})
	}

	for _, e := range entries {
		if e.valueVu == nil {
			continue
		}
		currentVu := e.valueVu
		for {
			var nextIdx = -1
			for i, u := range located {
				if consumed[i] || u.index <= currentVu.index {
					continue
				}
				if isContinuationLine(currentVu, u, limits, spec) {
					if nextIdx == -1 || u.index < located[nextIdx].index {
						nextIdx = i
					}
				}
			}
			if nextIdx == -1 {
				break
			}
			nextVu := located[nextIdx]
			e.ann.Text += strings.TrimSpace(nextVu.text)
			consumed[nextIdx] = true
			currentVu = nextVu
		}
	}

	for idx, u := range located {
		if consumed[idx] {
			continue
		}
		entries = append(entries, &resultEntry{
			ann: &sourcedocument.Annotation{Text: strings.TrimSpace(u.text)},
			x:   u.location.X,
			y:   u.location.Y,
		})
	}

	for _, ann := range preserved {
		entries = append(entries, &resultEntry{ann: ann})
	}

	sortEntriesByPosition(entries)

	converted := make([]*sourcedocument.Annotation, 0, len(entries))
	for _, e := range entries {
		if e == nil || e.ann == nil || strings.TrimSpace(e.ann.Text) == "" {
			continue
		}
		converted = append(converted, e.ann)
	}
	source.Annotations = converted
	return source, nil
}

func resolveSpec(n *Normalizer) *Spec {
	if n == nil || n.Spec == nil || n.Spec.KeyMaxRunes == 0 {
		return DefaultSpec()
	}
	return n.Spec
}

func splitAnnotations(annotations []*sourcedocument.Annotation) ([]*unit, []*sourcedocument.Annotation) {
	located := make([]*unit, 0, len(annotations))
	preserved := make([]*sourcedocument.Annotation, 0, len(annotations))
	for i, ann := range annotations {
		if ann == nil {
			continue
		}
		if ann.Location == nil {
			preserved = append(preserved, &sourcedocument.Annotation{Label: strings.TrimSpace(ann.Label), Text: strings.TrimSpace(ann.Text)})
			continue
		}
		text := strings.TrimSpace(ann.Text)
		if text == "" {
			continue
		}
		located = append(located, &unit{index: i, text: text, location: ann.Location})
	}
	return located, preserved
}

func sortUnitsByPosition(units []*unit) {
	sort.SliceStable(units, func(i, j int) bool {
		if units[i].location.Y != units[j].location.Y {
			return units[i].location.Y < units[j].location.Y
		}
		return units[i].location.X < units[j].location.X
	})
}

func sortEntriesByPosition(entries []*resultEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].y != entries[j].y {
			return entries[i].y < entries[j].y
		}
		return entries[i].x < entries[j].x
	})
}

func calculateThresholds(located []*unit, spec *Spec) thresholds {
	medianHeight := maxInt(medianDimension(located, func(u *unit) int { return u.location.H }), 1)
	medianWidth := maxInt(medianDimension(located, func(u *unit) int { return u.location.W }), 1)
	return thresholds{
		sameLineY: int(float64(medianHeight) * spec.SameLineYFactor),
		rightDist: int(float64(medianWidth) * spec.RightDistanceMult),
		belowDist: int(float64(medianHeight) * spec.BelowDistanceMult),
		alignX:    int(float64(medianWidth) * spec.AlignXTolerance),
	}
}

func medianDimension(units []*unit, get func(*unit) int) int {
	if len(units) == 0 {
		return 1
	}
	vals := make([]int, 0, len(units))
	for _, u := range units {
		v := get(u)
		if v <= 0 {
			continue
		}
		vals = append(vals, v)
	}
	if len(vals) == 0 {
		return 1
	}
	sort.Ints(vals)
	return vals[len(vals)/2]
}

func tryDirectColonSplit(text string) (bool, string, string) {
	norm := normalizeColon(strings.TrimSpace(text))
	idx := strings.Index(norm, ":")
	if idx < 0 {
		return false, "", ""
	}
	key := strings.TrimSpace(norm[:idx])
	value := strings.TrimSpace(norm[idx+1:])
	if key == "" || value == "" {
		return false, "", ""
	}
	return true, key, value
}

func normalizeColon(text string) string {
	replacer := strings.NewReplacer("：", ":", "﹕", ":", "︰", ":")
	return replacer.Replace(text)
}

func normalizeKey(text string) string {
	norm := normalizeColon(strings.TrimSpace(text))
	norm = strings.TrimSuffix(norm, ":")
	return strings.TrimSpace(norm)
}

func isValidDirectKV(spec *Spec, key, value string) bool {
	if !isKeyCandidate(spec, key) {
		return false
	}
	if isTimeOnly(key) || isRatioLike(key) {
		return false
	}
	if isDateTimeKey(key) {
		return looksLikeDateOrDateTime(value)
	}
	if isAmountKey(key) {
		return hasDigit(value)
	}
	return value != ""
}

func isKeyCandidate(spec *Spec, text string) bool {
	norm := normalizeKey(text)
	if norm == "" {
		return false
	}
	runes := countRunes(norm)
	if runes < 1 {
		return false
	}
	if runes > spec.KeyMaxRunes {
		return false
	}
	if strings.HasSuffix(strings.TrimSpace(text), ":") || strings.HasSuffix(strings.TrimSpace(text), "：") {
		return true
	}
	for _, kw := range spec.KeyWords {
		if norm == kw {
			return true
		}
	}
	for _, suffix := range spec.KeySuffixes {
		if strings.HasSuffix(norm, suffix) {
			return true
		}
	}
	if looksLikeValueToken(norm) {
		return false
	}
	return hasLetterLike(norm)
}

func looksLikeValueToken(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return true
	}
	if looksLikeDateOrDateTime(trimmed) || isTimeOnly(trimmed) || isRatioLike(trimmed) {
		return true
	}
	digits := 0
	total := 0
	for _, r := range trimmed {
		if unicode.IsSpace(r) {
			continue
		}
		total++
		if unicode.IsDigit(r) {
			digits++
		}
	}
	if total > 0 && digits*2 >= total {
		return true
	}
	return false
}

func hasLetterLike(text string) bool {
	for _, r := range text {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func countRunes(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

func horizontalScore(key, value *unit, sameLineYThreshold, rightDistanceThreshold int) (float64, bool) {
	kcy := key.location.Y + key.location.H/2
	vcy := value.location.Y + value.location.H/2
	yDiff := abs(kcy - vcy)
	if yDiff > sameLineYThreshold {
		return 0, false
	}
	dx := value.location.X - (key.location.X + key.location.W)
	if dx < 0 || dx > rightDistanceThreshold {
		return 0, false
	}
	xScore := 1.0 - clamp01(float64(dx)/float64(maxInt(rightDistanceThreshold, 1)))
	yScore := 1.0 - clamp01(float64(yDiff)/float64(maxInt(sameLineYThreshold, 1)))
	return xScore*0.5 + yScore*0.5, true
}

func belowScore(key, value *unit, belowDistanceThreshold, alignXTolerance int) (float64, bool) {
	dy := value.location.Y - (key.location.Y + key.location.H)
	if dy < 0 || dy > belowDistanceThreshold {
		return 0, false
	}
	if abs(key.location.X-value.location.X) > alignXTolerance {
		return 0, false
	}
	return 1.0 - clamp01(float64(dy)/float64(maxInt(belowDistanceThreshold, 1))), true
}

func semanticScore(keyText, valueText string) float64 {
	if strings.TrimSpace(valueText) == "" {
		return 0
	}
	if isDateTimeKey(keyText) && looksLikeDateOrDateTime(valueText) {
		return 1
	}
	if isAmountKey(keyText) && hasDigit(valueText) {
		return 1
	}
	if !isDateTimeKey(keyText) && !isAmountKey(keyText) && countRunes(strings.TrimSpace(valueText)) >= 2 {
		return 0.9
	}
	if hasDigit(valueText) {
		return 0.8
	}
	if countRunes(valueText) >= 2 {
		return 0.75
	}
	return 0.4
}

func patternScore(keyText, valueText string) float64 {
	if isDateTimeKey(keyText) {
		if looksLikeDateOrDateTime(valueText) {
			return 1
		}
		if countRunes(strings.TrimSpace(valueText)) >= 2 {
			return 0.7
		}
		return 0.4
	}
	if isAmountKey(keyText) {
		if hasDigit(valueText) {
			return 1
		}
		if countRunes(strings.TrimSpace(valueText)) >= 2 {
			return 0.7
		}
		return 0.4
	}
	if !isDateTimeKey(keyText) && !isAmountKey(keyText) && countRunes(strings.TrimSpace(valueText)) >= 2 {
		return 0.8
	}
	if hasDigit(valueText) {
		return 0.8
	}
	return 0.6
}

func looksLikeDateOrDateTime(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" {
		return false
	}
	if strings.Contains(v, "年") && strings.Contains(v, "月") && strings.Contains(v, "日") {
		return true
	}
	if strings.Contains(v, "-") || strings.Contains(v, "/") {
		if hasDigit(v) {
			return true
		}
	}
	return false
}

func isDateTimeKey(k string) bool {
	k = normalizeKey(k)
	return strings.Contains(k, "日期") || strings.Contains(k, "时间")
}

func isAmountKey(k string) bool {
	k = normalizeKey(k)
	return strings.Contains(k, "金额") || strings.Contains(k, "应付") || strings.Contains(k, "实付") || strings.Contains(k, "合计")
}

func isTimeOnly(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" {
		return false
	}
	parts := strings.Split(v, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, r := range p {
			if !unicode.IsDigit(r) {
				return false
			}
		}
	}
	return true
}

func isRatioLike(v string) bool {
	v = strings.TrimSpace(v)
	parts := strings.Split(v, ":")
	if len(parts) != 2 {
		return false
	}
	return allDigits(parts[0]) && allDigits(parts[1])
}

func allDigits(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func hasDigit(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func isContinuationLine(current, candidate *unit, limits thresholds, spec *Spec) bool {
	xDiff := abs(current.location.X - candidate.location.X)
	if xDiff > limits.alignX {
		return false
	}

	cy1 := current.location.Y + current.location.H/2
	cy2 := candidate.location.Y + candidate.location.H/2
	dy := abs(cy1 - cy2)

	if dy <= limits.sameLineY {
		return false
	}

	if dy > limits.belowDist {
		return false
	}

	if isKeyCandidate(spec, candidate.text) {
		// Verify if it is a strong key match (e.g. contains colon, explicit keyword or suffix)
		// and avoid merging only in those definitive cases, as generic key detection might just
		// be triggered by Chinese characters.
		norm := normalizeKey(candidate.text)
		if strings.HasSuffix(strings.TrimSpace(candidate.text), ":") || strings.HasSuffix(strings.TrimSpace(candidate.text), "：") {
			return false
		}
		for _, kw := range spec.KeyWords {
			if norm == kw {
				return false
			}
		}
		for _, suffix := range spec.KeySuffixes {
			if strings.HasSuffix(norm, suffix) {
				return false
			}
		}
	}

	return true
}
