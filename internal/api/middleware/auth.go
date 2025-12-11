package middleware

import (
	"context"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rohits-web03/obscyra/internal/config"
	"github.com/rohits-web03/obscyra/internal/utils"
)

type contextKey string

const UserIDKey contextKey = "userID"

var jwtSecret = config.Envs.JWTSecret

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		tokenStr, err := r.Cookie("token")
		if err != nil {
			utils.JSONResponse(w, http.StatusUnauthorized, utils.Payload{
				Success: false,
				Message: "Unauthorized",
			})
			return
		}

		token, err := jwt.Parse(tokenStr.Value, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			utils.JSONResponse(w, http.StatusUnauthorized, utils.Payload{
				Success: false,
				Message: "Unauthorized",
			})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			utils.JSONResponse(w, http.StatusUnauthorized, utils.Payload{
				Success: false,
				Message: "Unauthorized",
			})
			return
		}

		userID, ok := claims["userId"].(string)
		if !ok || userID == "" {
			utils.JSONResponse(w, http.StatusUnauthorized, utils.Payload{
				Success: false,
				Message: "Unauthorized",
			})
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
