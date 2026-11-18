package positions

import "time"

type realTicker struct{ ticker *time.Ticker }

func NewTicker(interval time.Duration) Ticker {
	return &realTicker{ticker: time.NewTicker(interval)}
}

func (t *realTicker) C() <-chan time.Time { return t.ticker.C }

func (t *realTicker) Stop() { t.ticker.Stop() }
