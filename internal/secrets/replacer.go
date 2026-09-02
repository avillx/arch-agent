package secrets

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Replacer struct {
	secretsSvc *Service
	replacer   *strings.Replacer

	mu sync.RWMutex
}

func NewReplacer(s *Service) *Replacer {
	r := &Replacer{
		secretsSvc: s,
	}

	r.Reload()

	return r
}

func (r *Replacer) Reload() {

	// lock fill service. If secrets is updated, new secrets can leak without lock
	r.mu.Lock()
	defer r.mu.Unlock()

	all := r.secretsSvc.All()

	keys := make([]string, 0, len(all))
	for k := range all {
		keys = append(keys, k)
	}

	// sort is necceccary cause sorter secret can hide a part of longer
	sort.Slice(keys, func(a, b int) bool {
		firstKey := keys[a]
		secondKey := keys[b]

		firstValue := all[firstKey]
		secondValue := all[secondKey]

		return len(firstValue) > len(secondValue)
	})

	pairs := make([]string, 0, len(keys)*2)
	for _, k := range keys {

		// suuuuper edge case
		if all[k] == "" {
			continue
		}

		secret := all[k]
		placeholder := fmt.Sprintf("{ env.%s }", k)

		pairs = append(pairs, secret, placeholder)
	}

	r.replacer = strings.NewReplacer(pairs...)
}

func (r *Replacer) Replace(text string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.replacer.Replace(text)
}
