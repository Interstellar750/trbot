package contain

import (
	"strings"

	"github.com/go-telegram/bot/models"
)

func Int(target int, candidates ...int) bool {
	for _, candidate := range candidates {
		if target == candidate {
			return true
		}
	}
	return false
}

func Int64(target int64, candidates ...int64) bool {
	for _, candidate := range candidates {
		if target == candidate {
			return true
		}
	}
	return false
}

func String(target string, candidates ...string) bool {
	for _, candidate := range candidates {
		if target == candidate {
			return true
		}
	}
	return false
}

func StringEqualFold(target string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.EqualFold(target, candidate) {
			return true
		}
	}
	return false
}

func SubString(target string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(candidate, target) {
			return true
		}
	}
	return false
}

func SubStringCaseInsensitive(target string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(strings.ToLower(candidate), strings.ToLower(target)) {
			return true
		}
	}
	return false
}

func ChatType(target models.ChatType, candidates ...models.ChatType) bool {
	for _, candidate := range candidates {
		if target == candidate {
			return true
		}
	}
	return false
}
