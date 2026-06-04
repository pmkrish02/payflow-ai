package middleware

import(
	"net/http"
	"github.com/golang-jwt/jwt/v5"
	"context"
)

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        
		tokenString := r.Header.Get("Authorization")
		if len(tokenString) > 7 && tokenString[:7] == "Bearer "{
			 tokenString = tokenString[7:]
			}
		if tokenString == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte("your-security-key"), nil
		})
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		claims := token.Claims.(jwt.MapClaims)
		userID := claims["sub"].(string)
		ctx := context.WithValue(r.Context(), "userID", userID)
		r = r.WithContext(ctx)
		next(w, r)
        
    }
}
