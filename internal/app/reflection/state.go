package reflection

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"
)

type EmoState map[string]float32

func (s EmoState) AugmentationFull() string {
	var sb strings.Builder
	for k, v := range s {
		sb.WriteString(fmt.Sprintf("%s: %d\n", k, int(v)))
	}
	return sb.String()
}

func (s EmoState) AugmentationTop() string {
	top := s.top(3)

	var sb strings.Builder
	for _, t := range top {
		if s[t] > 5 {
			prefix := LevelPrefix(s[t])
			newString := fmt.Sprintf("%s %s\n", prefix, t)
			sb.WriteString(newString)
		}
	}

	return sb.String()
}

func (s EmoState) top(n int) []string {
	keys := slices.Collect(maps.Keys(s))
	slices.SortFunc(keys, func(a, b string) int {
		return cmp.Compare(s[b], s[a])
	})
	top := keys[:min(n, len(keys))]
	return slices.DeleteFunc(top, func(elem string) bool {
		return s[elem] <= 15
	})
}

func LevelPrefix(level float32) string {
	switch {
	case 70 < level:
		return "extremly"
	case 50 < level && level < 70:
		return "very"
	case 25 < level && level < 50:
		return "moderate"
	case 10 < level && level < 25:
		return "slightly"
	default:
		return "a little bit"
	}
}
