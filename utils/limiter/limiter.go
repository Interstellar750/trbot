package limiter

import "sync"

var CallbackQuery = &Limiter{
	active: make(map[int64]int),
	limit:  2,
}

type Limiter struct {
	mu     sync.Mutex
	active map[int64]int // 用户ID -> 当前正在处理的操作
	limit  int
}

func New(limit int) *Limiter {
	return &Limiter{
		active: make(map[int64]int),
		limit:  limit,
	}
}

func (ul *Limiter) Try(userID int64) bool {
	ul.mu.Lock()
	defer ul.mu.Unlock()
	if ul.active[userID] >= ul.limit {
		return false
	}
	ul.active[userID]++
	return true
}

func (ul *Limiter) Release(userID int64) {
	ul.mu.Lock()
	defer ul.mu.Unlock()
	if ul.active[userID] > 0 {
		ul.active[userID]--
	}
}
