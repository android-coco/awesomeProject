package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/jwt"
	"time"
)

var (
	sigKey = []byte("signature_hmac_secret_shared_key")
	encKey = []byte("GCM_AES_256_secret_shared_key_32")
)

type fooClaims struct {
	Foo string `json:"foo"`
}

func main() {
	app := iris.New()

	signer := jwt.NewSigner(jwt.HS256, sigKey, 10*time.Minute)
	// Enable payload encryption with:
	// signer.WithEncryption(encKey, nil)
	app.Get("/", generateToken(signer))

	verifier := jwt.NewVerifier(jwt.HS256, sigKey)
	// Enable server-side token block feature (even before its expiration time):
	verifier.WithDefaultBlocklist()
	// Enable payload decryption with:
	// verifier.WithDecryption(encKey, nil)
	verifyMiddleware := verifier.Verify(func() interface{} {
		return new(fooClaims)
	})

	protectedAPI := app.Party("/protected")
	// Register the verify middleware to allow access only to authorized clients.
	protectedAPI.Use(verifyMiddleware)
	// ^ or UseRouter(verifyMiddleware) to disallow unauthorized http error handlers too.

	protectedAPI.Get("/", protected)
	// Invalidate the token through server-side, even if it's not expired yet.
	protectedAPI.Get("/logout", logout)

	// http://localhost:8888
	// http://localhost:8888/protected?token=$token (or Authorization: Bearer $token)
	// http://localhost:8888/protected/logout?token=$token
	// http://localhost:8888/protected?token=$token (401)
	_ = app.Listen(":8888")
}

func generateToken(signer *jwt.Signer) iris.Handler {
	return func(ctx iris.Context) {
		claims := fooClaims{Foo: "bar"}

		token, err := signer.Sign(claims)
		if err != nil {
			ctx.StopWithStatus(iris.StatusInternalServerError)
			return
		}

		_, _ = ctx.Write(token)
	}
}

func protected(ctx iris.Context) {
	// Get the verified and decoded claims.
	claims := jwt.Get(ctx).(*fooClaims)

	// Optionally, get token information if you want to work with them.
	// Just an example on how you can retrieve all the standard claims (set by signer's max age, "exp").
	standardClaims := jwt.GetVerifiedToken(ctx).StandardClaims
	expiresAtString := standardClaims.ExpiresAt().Format(ctx.Application().ConfigurationReadOnly().GetTimeFormat())
	timeLeft := standardClaims.Timeleft()

	_, _ = ctx.HTML("foo=%s\nexpires at: %s\ntime left: %s\n", claims.Foo, expiresAtString, timeLeft)
}

func logout(ctx iris.Context) {
	err := ctx.Logout()
	if err != nil {
		_, _ = ctx.WriteString(err.Error())
	} else {
		_, _ = ctx.HTML("token invalidated, a new token is required to access the protected API")
	}
}
