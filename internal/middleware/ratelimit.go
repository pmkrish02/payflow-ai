package middleware

import(
	"net/http"
	"github.com/redis/go-redis/v9"
	"time"
	)


func RateLimitMiddleware(rdb *redis.Client) func(http.HandlerFunc) http.HandlerFunc {

    return func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
            userID := ctx.Value("userID").(string)
			rate, err := rdb.Incr(ctx, "rate:"+userID).Result()
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			if rate == 1{
				rdb.Expire(ctx, "rate:"+userID, time.Minute)
			}
			if rate > 100{
				http.Error(w,"rate Limit exceeded",http.StatusTooManyRequests)
				return
			}			
			next(w,r)
        }
    }
}