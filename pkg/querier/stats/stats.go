package stats

import (
	"context"
	"net/http"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/weaveworks/common/user"
)

type contextKey int

var ctxKey = contextKey(0)

// FromContext gets the Stats out of the Context.
func FromContext(ctx context.Context) *Stats {
	o := ctx.Value(ctxKey)
	if o == nil {
		// To make user of this function's life easier, return an empty stats
		// if none is found in the context.
		return &Stats{}
	}
	return o.(*Stats)
}

// Stats for a single query.
type Stats struct {
	WallTime time.Duration
	Series   int
	Samples  int64
}

// Merge the provide Stats into this one.
func (s *Stats) Merge(other *Stats) {
	s.WallTime += other.WallTime
	s.Series += other.Series
	s.Samples += other.Samples
}

// Middleware initialises the stats in the request context, records wall clock time
// and logs the results.
type Middleware struct {
	logger log.Logger
}

// NewMiddleware makes a new Middleware.
func NewMiddleware(logger log.Logger) Middleware {
	return Middleware{
		logger: logger,
	}
}

// Wrap implements middleware.Interface.
func (m Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := user.ExtractOrgID(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		start := time.Now()
		stats := &Stats{}
		r = r.WithContext(context.WithValue(r.Context(), ctxKey, stats))

		defer func() {
			stats.WallTime = time.Since(start)
			level.Info(m.logger).Log(
				"usedID", userID,
				"time", stats.WallTime,
				"series", stats.Series,
				"samples", stats.Samples,
			)
		}()

		next.ServeHTTP(w, r)
	})
}