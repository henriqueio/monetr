package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	locale "github.com/elliotcourant/go-lclocale"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
	"github.com/monetr/monetr/server/communication"
	"github.com/monetr/monetr/server/crumbs"
	"github.com/monetr/monetr/server/models"
	"github.com/monetr/monetr/server/repository"
	"github.com/monetr/monetr/server/security"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const ClearAuthentication = ""

// updateAuthenticationCookie is used to maintain the authentication credentials that the client uses to communicate
// with the API. When this is called with a token that token will be returned to the client in the response as a
// Set-Cookie header. If a blank token is provided then the cookie is updated to expire immediately and the value of the
// cookie is set to blank.
func (c *Controller) updateAuthenticationCookie(ctx echo.Context, token string) {
	sameSite := http.SameSiteDefaultMode
	if c.configuration.Server.Cookies.SameSiteStrict {
		sameSite = http.SameSiteStrictMode
	}

	expiration := c.clock.Now().AddDate(0, 0, 14)
	if token == "" {
		expiration = c.clock.Now().Add(-1 * time.Second)
	}

	if c.configuration.Server.Cookies.Name == "" {
		panic("authentication cookie name is blank")
	}

	hostname := c.configuration.APIDomainName
	parts := strings.SplitN(c.configuration.APIDomainName, ":", 2)
	if len(parts) > 1 {
		hostname = parts[0]
	}

	ctx.SetCookie(&http.Cookie{
		Name:     c.configuration.Server.Cookies.Name,
		Value:    token,
		Path:     "/",
		Domain:   hostname,
		Expires:  expiration,
		MaxAge:   0,
		Secure:   c.configuration.GetHTTPSecureCookie(),
		HttpOnly: true,
		SameSite: sameSite,
	})
}

func (c *Controller) postLogin(ctx echo.Context) error {
	var loginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Captcha  string `json:"captcha"`
		TOTP     string `json:"totp"`
		IsMobile bool   `json:"isMobile"`
	}
	if err := ctx.Bind(&loginRequest); err != nil {
		return c.wrapAndReturnError(ctx, err, http.StatusBadRequest, "malformed json")
	}

	// This will take the captcha from the request and validate it if the API is
	// configured to do so. If it is enabled and the captcha fails then an error
	// is returned to the client.

	if err := c.validateLoginCaptcha(c.getContext(ctx), loginRequest.Captcha); err != nil {
		return c.wrapAndReturnError(ctx, err, http.StatusBadRequest, "valid ReCAPTCHA is required")
	}

	loginRequest.Email = strings.ToLower(strings.TrimSpace(loginRequest.Email))
	loginRequest.Password = strings.TrimSpace(loginRequest.Password)
	loginRequest.TOTP = strings.TrimSpace(loginRequest.TOTP)

	if err := c.validateLogin(ctx, loginRequest.Email, loginRequest.Password); err != nil {
		return err // Validate login errors are valid http errors.
	}

	secureRepo := c.mustGetSecurityRepository(ctx)
	login, requiresPasswordChange, err := secureRepo.Login(c.getContext(ctx), loginRequest.Email, loginRequest.Password)
	switch errors.Cause(err) {
	case repository.ErrInvalidCredentials:
		return c.returnError(ctx, http.StatusUnauthorized, "invalid email and password")
	case nil:
		// If no error was returned then do nothing.
		break
	default:
		return c.wrapPgError(ctx, err, "failed to authenticate")
	}

	// I want to track how many of these types of things we get.
	crumbs.AddTag(c.getContext(ctx), "requiresPasswordChange", fmt.Sprint(requiresPasswordChange))

	log := c.getLog(ctx).WithField("loginId", login.LoginId)

	// If we want to verify emails and the login does not have a verified email address, then return an error to the
	// user.
	if c.configuration.Email.ShouldVerifyEmails() && !login.IsEmailVerified {
		log.Debug("login email address is not verified, please verify before continuing")
		return c.failure(ctx, http.StatusPreconditionRequired, EmailNotVerifiedError{})
	}

	if requiresPasswordChange {
		// If the server is not configured to allow password resets return an error.
		if !c.configuration.Email.AllowPasswordReset() {
			return c.returnError(ctx, http.StatusNotAcceptable, "login requires password reset, but password reset is not allowed")
		}

		passwordResetToken, err := c.clientTokens.Create(
			security.ResetPasswordAudience,
			5*time.Minute, // Use a much shorter lifetime than usually would be configured.
			security.Claims{
				EmailAddress: login.Email,
				UserId:       "",
				AccountId:    "",
				LoginId:      login.LoginId.String(),
			},
		)
		if err != nil {
			return c.wrapAndReturnError(ctx, err, http.StatusInternalServerError, "Failed to generate a password reset token")
		}

		log.Info("login requires a password change")
		return c.failure(ctx, http.StatusPreconditionRequired, PasswordResetRequiredError{
			ResetToken: passwordResetToken,
		})
	}

	// Check if the login requires MFA in order to authenticate.
	if login.TOTP != "" && loginRequest.TOTP == "" {
		log.Debug("login requires TOTP MFA, but none was provided")
		return c.failure(ctx, http.StatusPreconditionRequired, MFARequiredError{})
	} else if login.TOTP != "" && loginRequest.TOTP != "" {
		// If the login does require TOTP and a code was provided in the request, then validate that the provided code is
		// correct.
		log.Trace("login requires TOTP MFA, and a code was provided; it will be verified")

		if err := login.VerifyTOTP(loginRequest.TOTP, c.clock.Now()); err != nil {
			log.Trace("provided TOTP MFA code is not valid")
			return c.returnError(ctx, http.StatusUnauthorized, "invalid TOTP code")
		}

		log.Trace("provided TOTP MFA code is valid")
	} else if login.TOTP == "" && loginRequest.TOTP != "" {
		log.Warn("login does not require TOTP MFA, but a code was provided anyway")
	}

	switch len(login.Users) {
	case 0:
		// TODO (elliotcourant) Should we allow them to create an account?
		return c.returnError(ctx, http.StatusInternalServerError, "user has no accounts")
	case 1:
		user := login.Users[0]

		crumbs.IncludeUserInScope(c.getContext(ctx), user.AccountId)
		token, err := c.clientTokens.Create(security.AuthenticatedAudience, 14*24*time.Hour, security.Claims{
			EmailAddress: login.Email,
			UserId:       user.UserId.String(),
			AccountId:    user.AccountId.String(),
			LoginId:      user.LoginId.String(),
		})
		if err != nil {
			return c.wrapAndReturnError(ctx, err, http.StatusInternalServerError, "could not generate token")
		}

		result := map[string]interface{}{
			"isActive": true,
		}

		if !loginRequest.IsMobile {
			c.updateAuthenticationCookie(ctx, token)
		} else {
			result["token"] = token
		}

		if !c.configuration.Stripe.IsBillingEnabled() {
			// Return their account token.
			return ctx.JSON(http.StatusOK, result)
		}

		subscriptionIsActive, err := c.paywall.GetSubscriptionIsActive(c.getContext(ctx), user.AccountId)
		if err != nil {
			return c.wrapAndReturnError(ctx, err, http.StatusInternalServerError, "failed to determine whether or not subscription is active")
		}

		result["isActive"] = subscriptionIsActive

		if !subscriptionIsActive {
			result["nextUrl"] = "/account/subscribe"
		}

		return ctx.JSON(http.StatusOK, result)
	default:
		// If the login has more than one user then we want to generate a temp
		// JWT that will only grant them access to API endpoints not specific to
		// an account.
		return c.badRequest(ctx, "multiple accounts not implemented, please contact support")
	}
}

func (c *Controller) logoutEndpoint(ctx echo.Context) error {
	if _, err := ctx.Cookie(c.configuration.Server.Cookies.Name); err == http.ErrNoCookie {
		return ctx.NoContent(http.StatusOK)
	}

	c.updateAuthenticationCookie(ctx, ClearAuthentication)
	return ctx.NoContent(http.StatusOK)
}

func (c *Controller) postRegister(ctx echo.Context) error {
	if !c.configuration.AllowSignUp {
		return c.notFound(ctx, "sign up is not enabled on this server")
	}

	var registerRequest struct {
		Email     string  `json:"email"`
		Password  string  `json:"password"`
		FirstName string  `json:"firstName"`
		LastName  string  `json:"lastName"`
		Timezone  string  `json:"timezone"`
		Locale    string  `json:"locale"`
		Captcha   string  `json:"captcha"`
		BetaCode  *string `json:"betaCode"`
	}
	if err := ctx.Bind(&registerRequest); err != nil {
		return c.invalidJson(ctx)
	}

	// This will take the captcha from the request and validate it if the API is
	// configured to do so. If it is enabled and the captcha fails then an error
	// is returned to the client.
	if err := c.validateRegistrationCaptcha(c.getContext(ctx), registerRequest.Captcha); err != nil {
		return c.wrapAndReturnError(ctx, err, http.StatusBadRequest, "valid ReCAPTCHA is required")
	}

	log := c.getLog(ctx)

	registerRequest.Email = strings.TrimSpace(registerRequest.Email)
	registerRequest.Password = strings.TrimSpace(registerRequest.Password)
	registerRequest.FirstName = strings.TrimSpace(registerRequest.FirstName)
	registerRequest.Locale = strings.TrimSpace(registerRequest.Locale)
	if registerRequest.BetaCode != nil {
		*registerRequest.BetaCode = strings.TrimSpace(*registerRequest.BetaCode)
	}

	if registerRequest.Locale == "" {
		return c.badRequest(ctx, "Locale must be specified to register")
	}

	if !locale.Valid(registerRequest.Locale) {
		return c.badRequest(ctx, "Invalid or unrecognized locale")
	}

	if err := c.validateRegistration(
		ctx,
		registerRequest.Email,
		registerRequest.Password,
		registerRequest.FirstName,
	); err != nil {
		return err // validateRegistration also returns a valid http error that can just be passed through.
	}

	timezone, err := time.LoadLocation(registerRequest.Timezone)
	if err != nil {
		return c.wrapAndReturnError(ctx, err, http.StatusBadRequest, "failed to parse timezone")
	}

	// If the registration details provided look good then we want to create an
	// unauthenticated repo. This will give us some basic database access
	// without being able to access user information directly. It is essentially
	// a write only interface to the database.
	repo, err := c.getUnauthenticatedRepository(ctx)
	if err != nil {
		return c.wrapAndReturnError(ctx, err, http.StatusInternalServerError,
			"cannot register user",
		)
	}

	var beta *models.Beta
	if c.configuration.Beta.EnableBetaCodes {
		if registerRequest.BetaCode == nil || *registerRequest.BetaCode == "" {
			return c.badRequest(ctx, "beta code required for registration")
		}

		beta, err = repo.ValidateBetaCode(c.getContext(ctx), *registerRequest.BetaCode)
		if err != nil {
			return c.wrapPgError(ctx, err, "could not verify beta code")
		}
	}

	// Create the user's login record in the database, this will return the login
	// record including the new login's loginId which we will need below.
	login, err := repo.CreateLogin(
		c.getContext(ctx),
		registerRequest.Email,
		registerRequest.Password,
		registerRequest.FirstName,
		registerRequest.LastName,
	)
	if err != nil {
		switch errors.Cause(err) {
		case repository.ErrEmailAlreadyExists:
			return c.failure(ctx, http.StatusBadRequest, EmailAlreadyExists{})
		default:
			return c.wrapAndReturnError(ctx, err, http.StatusInternalServerError,
				"failed to create login",
			)
		}
	}
	log = log.WithField("loginId", login.LoginId)

	var trialEndsAt *time.Time
	if c.configuration.Stripe.IsBillingEnabled() {
		expiration := c.clock.Now().AddDate(0, 0, c.configuration.Stripe.FreeTrialDays)
		log.WithFields(logrus.Fields{
			"trialDays":   c.configuration.Stripe.FreeTrialDays,
			"trialEndsAt": expiration,
		}).Debug("billing is enabled, new account for login will be on a trial")

		trialEndsAt = &expiration
	}

	account := models.Account{
		Timezone:    timezone.String(),
		TrialEndsAt: trialEndsAt,
		Locale:      registerRequest.Locale,
	}
	// Now that the login exists we can create the account, at the time of
	// writing this we are only using the local time zone of the server, but in
	// the future I want to have it somehow use the user's timezone.
	if err = repo.CreateAccountV2(c.getContext(ctx), &account); err != nil {
		return c.wrapAndReturnError(ctx, err, http.StatusInternalServerError,
			"failed to create account",
		)
	}

	crumbs.IncludeUserInScope(c.getContext(ctx), account.AccountId)

	user := models.User{
		LoginId:   login.LoginId,
		AccountId: account.AccountId,
	}

	// Now that we have an accountId we can create the user object which will
	// bind the login and the account together.
	err = repo.CreateUser(
		c.getContext(ctx),
		login.LoginId,
		account.AccountId,
		&user,
	)
	if err != nil {
		return c.wrapAndReturnError(ctx, err, http.StatusInternalServerError,
			"failed to create user",
		)
	}

	if beta != nil {
		if err = repo.UseBetaCode(c.getContext(ctx), beta.BetaId, user.UserId); err != nil {
			return c.wrapAndReturnError(ctx, err, http.StatusInternalServerError,
				"failed to use beta code",
			)
		}
	}

	// If SMTP is enabled and we are verifying emails then we want to create a
	// registration record and send the user a verification email.
	if c.configuration.Email.ShouldVerifyEmails() {
		verificationToken, err := c.clientTokens.Create(
			security.VerifyEmailAudience,
			c.configuration.Email.Verification.TokenLifetime,
			security.Claims{
				EmailAddress: login.Email,
				UserId:       "",
				AccountId:    "",
				LoginId:      login.LoginId.String(),
			},
		)
		if err != nil {
			return c.wrapAndReturnError(
				ctx,
				err,
				http.StatusInternalServerError,
				"could not generate email verification token",
			)
		}

		if err = c.email.SendVerification(c.getContext(ctx), communication.VerifyEmailParams{
			BaseURL:      c.configuration.GetUIURL(),
			Email:        login.Email,
			FirstName:    login.FirstName,
			LastName:     login.LastName,
			SupportEmail: "support@monetr.app",
			VerifyURL:    fmt.Sprintf("%s/verify/email?token=%s", c.configuration.GetUIURL(), url.QueryEscape(verificationToken)),
		}); err != nil {
			return c.wrapAndReturnError(
				ctx,
				err,
				http.StatusInternalServerError,
				"failed to send verification email",
			)
		}

		return ctx.JSON(http.StatusOK, map[string]interface{}{
			"message":             "A verification email has been sent to your email address, please verify your email.",
			"requireVerification": true,
		})
	}

	// If we are not requiring email verification to activate an account we can
	// simply return a token here for the user to be signed in.
	token, err := c.clientTokens.Create(security.AuthenticatedAudience, 14*24*time.Hour, security.Claims{
		EmailAddress: login.Email,
		UserId:       user.UserId.String(),
		AccountId:    user.AccountId.String(),
		LoginId:      user.LoginId.String(),
	})
	if err != nil {
		return c.wrapAndReturnError(ctx, err, http.StatusInternalServerError,
			"failed to create token",
		)
	}

	user.Login = login
	user.Account = &account

	c.updateAuthenticationCookie(ctx, token)

	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"nextUrl":             "/setup",
		"requireVerification": false,
	})
}

func (c *Controller) verifyEndpoint(ctx echo.Context) error {
	if !c.configuration.Email.ShouldVerifyEmails() {
		return c.notFound(ctx, "email verification is not enabled")
	}

	var verifyRequest struct {
		Token string `json:"token"`
	}
	if err := ctx.Bind(&verifyRequest); err != nil {
		return c.invalidJson(ctx)
	}

	if strings.TrimSpace(verifyRequest.Token) == "" {
		return c.badRequest(ctx, "Token cannot be blank")
	}

	claims, err := c.clientTokens.Parse(security.VerifyEmailAudience, verifyRequest.Token)
	if err != nil {
		return c.wrapAndReturnError(ctx, err, http.StatusBadRequest, "Invalid email verification")
	}

	repo := c.mustGetUnauthenticatedRepository(ctx)
	if err := repo.SetEmailVerified(c.getContext(ctx), claims.EmailAddress); err != nil {
		return c.wrapAndReturnError(ctx, err, http.StatusBadRequest, "Invalid email verification")
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"nextUrl": "/login",
		"message": "Your email is now verified. Please login.",
	})
}

func (c *Controller) resendVerification(ctx echo.Context) error {
	log := c.getLog(ctx)
	if !c.configuration.Email.ShouldVerifyEmails() {
		return c.notFound(ctx, "email verification is not enabled")
	}

	var request struct {
		Email   string  `json:"email"`
		Captcha *string `json:"captcha"`
	}
	if err := ctx.Bind(&request); err != nil {
		return c.invalidJson(ctx)
	}

	request.Email = strings.TrimSpace(strings.ToLower(request.Email))
	if request.Email == "" {
		return c.badRequest(ctx, "email must be provided to resend verification link")
	}

	if c.configuration.ReCAPTCHA.Enabled {
		if request.Captcha == nil {
			return c.badRequest(ctx, "must provide ReCAPTCHA")
		}

		if err := c.validateCaptchaMaybe(c.getContext(ctx), *request.Captcha); err != nil {
			return c.wrapAndReturnError(ctx, err, http.StatusBadRequest, "invalid ReCAPTCHA provided")
		}
	}

	unauthedRepo := c.mustGetUnauthenticatedRepository(ctx)
	login, err := unauthedRepo.GetLoginForEmail(c.getContext(ctx), request.Email)
	if err != nil {
		log.WithError(err).Warn("failed to get login for email address to resend verification")
		return ctx.NoContent(http.StatusOK)
	}

	verificationToken, err := c.clientTokens.Create(
		security.VerifyEmailAudience,
		c.configuration.Email.ForgotPassword.TokenLifetime,
		security.Claims{
			EmailAddress: login.Email,
			UserId:       "",
			AccountId:    "",
			LoginId:      login.LoginId.String(),
		},
	)
	if err != nil {
		c.reportWrappedError(ctx, err, "failed to regenerate email verification token")
		return ctx.NoContent(http.StatusOK)
	}

	if err = c.email.SendVerification(c.getContext(ctx), communication.VerifyEmailParams{
		BaseURL:      c.configuration.GetUIURL(),
		Email:        login.Email,
		FirstName:    login.FirstName,
		LastName:     login.LastName,
		SupportEmail: "support@monetr.app",
		VerifyURL:    fmt.Sprintf("%s/verify/email?token=%s", c.configuration.GetUIURL(), url.QueryEscape(verificationToken)),
	}); err != nil {
		c.reportWrappedError(ctx, err, "failed to send (re-send) verification email")
		return ctx.NoContent(http.StatusOK)
	}

	return ctx.NoContent(http.StatusOK)
}

// Send Password Reset Link
// @Summary Send Password Reset Link
// @id send-password-reset-link
// @tags Authentication
// @description This endpoint should be used to send password reset links to clients who have forgotten their password.
// @Produce json
// @Accept json
// @Param Token body swag.ForgotPasswordRequest true "Forgot Password Request"
// @Router /authentication/forgot [post]
// @Success 200
// @Failure 400 {object} swag.ForgotPasswordBadRequest
// @Failure 428 {object} swag.ForgotPasswordEmailNotVerifiedError Email verification required.
// @Failure 500 {object} ApiError Something went wrong on our end.
func (c *Controller) sendForgotPassword(ctx echo.Context) error {
	if !c.configuration.Email.AllowPasswordReset() {
		return c.notFound(ctx, "password reset not enabled")
	}

	var sendForgotPasswordRequest struct {
		Email     string `json:"email"`
		ReCAPTCHA string `json:"captcha"`
	}
	if err := ctx.Bind(&sendForgotPasswordRequest); err != nil {
		return c.invalidJson(ctx)
	}

	// Clean up some of the input provided just in case.
	sendForgotPasswordRequest.Email = strings.TrimSpace(strings.ToLower(sendForgotPasswordRequest.Email))
	sendForgotPasswordRequest.ReCAPTCHA = strings.TrimSpace(sendForgotPasswordRequest.ReCAPTCHA)

	if sendForgotPasswordRequest.Email == "" {
		return c.badRequest(ctx, "Must provide an email address.")
	}

	// If we require ReCAPTCHA then make sure they provide it.
	if c.configuration.ReCAPTCHA.ShouldVerifyForgotPassword() {
		if sendForgotPasswordRequest.ReCAPTCHA == "" {
			return c.badRequest(ctx, "Must provide a valid ReCAPTCHA.")
		}

		if err := c.validateCaptchaMaybe(c.getContext(ctx), sendForgotPasswordRequest.ReCAPTCHA); err != nil {
			return c.wrapAndReturnError(ctx, err, http.StatusBadRequest, "Valid ReCAPTCHA is required")
		}
	}

	// We need to retrieve the login record for the email in order to verify that the login actually exists, as well as
	// make sure the login email address has been verified.
	login, err := c.mustGetUnauthenticatedRepository(ctx).GetLoginForEmail(
		c.getContext(ctx),
		sendForgotPasswordRequest.Email,
	)
	if err != nil {
		crumbs.Debug(c.getContext(ctx), "No password reset email will be sent, login for email not found", map[string]interface{}{
			"error": err.Error(),
		})
		// Don't return an error to the client, don't want them to know if it failed to send.
		return ctx.NoContent(http.StatusOK)
	}

	// If the login's email is not verified then return an error to the client.
	if c.configuration.Email.ShouldVerifyEmails() && !login.GetEmailIsVerified() {
		return c.returnError(
			ctx,
			http.StatusPreconditionRequired,
			"You must verify your email before you can send forgot password requests.",
		)
	}

	// Generate the password reset token.
	// TODO When we allow for email address to be changed, we will also need to verify that a token being used is for
	//  the same login that it was generated for. Otherwise someone could generate a token for an email, then change the
	//  email of the login -> change the email of another login to the first email; then reset the password for a
	//  different login. I don't think this is a security risk as the actor would need to have access to both logins to
	//  begin with. But it might cause some goofy issues and is definitely not desired behavior.
	passwordResetToken, err := c.clientTokens.Create(
		security.ResetPasswordAudience,
		c.configuration.Email.ForgotPassword.TokenLifetime,
		security.Claims{
			EmailAddress: login.Email,
			UserId:       "",
			AccountId:    "",
			LoginId:      login.LoginId.String(),
		},
	)
	if err != nil {
		return c.wrapAndReturnError(
			ctx,
			err,
			http.StatusInternalServerError,
			"Failed to generate a password reset token",
		)
	}

	if err = c.email.SendPasswordReset(c.getContext(ctx), communication.PasswordResetParams{
		BaseURL:      c.configuration.GetUIURL(),
		Email:        login.Email,
		FirstName:    login.FirstName,
		LastName:     login.LastName,
		SupportEmail: "support@monetr.app",
		ResetURL:     fmt.Sprintf("%s/password/reset?token=%s", c.configuration.GetUIURL(), url.QueryEscape(passwordResetToken)),
	}); err != nil {
		return c.wrapAndReturnError(
			ctx,
			err,
			http.StatusInternalServerError,
			"Failed to send password reset email",
		)
	}

	return ctx.NoContent(http.StatusOK)
}

// Reset Password
// @Summary Reset Password
// @id reset-password
// @tags Authentication
// @description This endpoint handles resetting passwords for users who have forgotten theirs. It requires a `token` be
// @description provided that comes from the email the `/authentication/forgot` endpoint sends to the login's email
// @description address.
// @Produce json
// @Accept json
// @Param Token body swag.ResetPasswordRequest true "Reset Password Request"
// @Router /authentication/reset [post]
// @Success 200
// @Failure 400 {object} swag.ResetPasswordBadRequest
// @Failure 500 {object} ApiError Something went wrong on our end.
func (c *Controller) resetPassword(ctx echo.Context) error {
	if !c.configuration.Email.AllowPasswordReset() {
		return c.notFound(ctx, "password reset not enabled")
	}

	var resetPasswordRequest struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := ctx.Bind(&resetPasswordRequest); err != nil {
		return c.invalidJson(ctx)
	}

	resetPasswordRequest.Token = strings.TrimSpace(resetPasswordRequest.Token)
	resetPasswordRequest.Password = strings.TrimSpace(resetPasswordRequest.Password)

	// The token is what verifies that the user is who they say they are even without a password. The token is emailed
	// to their verified email address.
	if resetPasswordRequest.Token == "" {
		return c.badRequest(ctx, "Token must be provided to reset password.")
	}

	if len(resetPasswordRequest.Password) < 8 {
		return c.badRequest(ctx, "Password must be at least 8 characters long.")
	}

	resetClaims, err := c.clientTokens.Parse(
		security.ResetPasswordAudience,
		resetPasswordRequest.Token,
	)
	if err != nil {
		return c.wrapAndReturnError(
			ctx,
			err,
			http.StatusBadRequest,
			"Failed to validate password reset token",
		)
	}

	unauthenticatedRepo := c.mustGetUnauthenticatedRepository(ctx)

	// Retrieve the login for the email address in the token.
	login, err := unauthenticatedRepo.GetLoginForEmail(c.getContext(ctx), resetClaims.EmailAddress)
	if err != nil {
		return c.wrapPgError(ctx, err, "Failed to verify login for email address")
	}

	// If the login's password has been changed since this token was issued, then this token is no longer valid. This
	// will basically make sure that a token cannot be used twice.
	if login.PasswordResetAt != nil && !login.PasswordResetAt.Before(resetClaims.CreatedAt) {
		return c.badRequest(
			ctx,
			"Password has already been reset, you must request another password reset link.",
		)
	}

	if err = unauthenticatedRepo.ResetPassword(
		c.getContext(ctx),
		login.LoginId,
		resetPasswordRequest.Password,
	); err != nil {
		return c.wrapPgError(ctx, err, "Failed to reset password")
	}

	if err := c.email.SendPasswordChanged(c.getContext(ctx), communication.PasswordChangedParams{
		BaseURL:      c.configuration.GetUIURL(),
		Email:        login.Email,
		FirstName:    login.FirstName,
		LastName:     login.LastName,
		SupportEmail: "support@monetr.app",
	}); err != nil {
		return c.wrapAndReturnError(
			ctx,
			err,
			http.StatusInternalServerError,
			"Failed to send password changed notification",
		)
	}

	return ctx.NoContent(http.StatusOK)
}

func (c *Controller) validateRegistration(ctx echo.Context, email, password, firstName string) error {
	if email == "" {
		return c.badRequest(ctx, "Email cannot be blank")
	}

	if len(password) < 8 {
		return c.badRequest(ctx, "Password must be at least 8 characters")
	}

	if firstName == "" {
		return c.badRequest(ctx, "First name cannot be left blank")
	}

	return nil
}

func (c *Controller) validateLoginCaptcha(ctx context.Context, captcha string) error {
	if !c.configuration.ReCAPTCHA.ShouldVerifyLogin() {
		// If it is disabled then we don't need to do anything.
		return nil
	}

	return c.validateCaptchaMaybe(ctx, captcha)
}

func (c *Controller) validateRegistrationCaptcha(ctx context.Context, captcha string) error {
	if !c.configuration.ReCAPTCHA.ShouldVerifyRegistration() {
		// If it is disabled then we don't need to do anything.
		return nil
	}

	return c.validateCaptchaMaybe(ctx, captcha)
}

func (c *Controller) validateCaptchaMaybe(ctx context.Context, captcha string) error {
	if captcha == "" {
		return errors.Errorf("captcha is not valid")
	}

	span := sentry.StartSpan(ctx, "ReCAPTCHA")
	defer span.Finish()

	return c.captcha.VerifyCaptcha(ctx, captcha)
}

// validateLogin takes the current request context and the provided email and password, it then validates thats the
// credentials are valid based on some basic constraints. The password must be so long and the email address provided
// must be at least a valid email formated string. If the credentials are not valid in this regard then an http error is
// returned and can be passed immediately back up through the controller.
func (c *Controller) validateLogin(ctx echo.Context, email, password string) error {
	if len(password) < 8 {
		return c.badRequest(ctx, "Password must be at least 8 characters")
	}

	address, err := mail.ParseAddress(email)
	if err != nil {
		return c.badRequest(ctx, "Email address provided is not valid")
	}

	if !strings.EqualFold(address.Address, email) {
		return c.badRequest(ctx, "Email address provided is not valid")
	}

	return nil
}
