package config

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ddvk/rmfakecloud/internal/email"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/pbkdf2"
)

const (
	// DefaultPort the default port
	DefaultPort = "3000"
	// DefaultDataDir default folder for storage
	DefaultDataDir = "data"

	// ReadStorageExpirationInMinutes time the token is valid
	ReadStorageExpirationInMinutes = 5
	// WriteStorageExpirationInMinutes time the token is valid
	WriteStorageExpirationInMinutes = 5

	// DefaultHost fake url
	DefaultHost = "local.appspot.com"

	// EnvLogLevel environment variable for the log level
	EnvLogLevel = "LOGLEVEL"
	// EnvLogFormat type of log format
	EnvLogFormat = "LOGFORMAT"
	// envDataDir
	envDataDir = "DATADIR"
	envPort    = "PORT"
	// EnvStorageURL the external name of the service
	EnvStorageURL = "STORAGE_URL"
	// envTLSCert the path of the cert file
	envTLSCert = "TLS_CERT"
	// envTLSKey the path of the private key
	envTLSKey = "TLS_KEY"

	// auth
	envJWTSecretKey     = "JWT_SECRET_KEY"
	envRegistrationOpen = "OPEN_REGISTRATION"

	// envSMTPServer the mail server
	envSMTPServer = "RM_SMTP_SERVER"
	// envSMTPUsername the username for the mail server
	envSMTPUsername = "RM_SMTP_USERNAME"
	// envSMTPPassword pass
	envSMTPPassword = "RM_SMTP_PASSWORD"
	// envSMTPHelo custom helo
	envSMTPHelo = "RM_SMTP_HELO"
	// no tls, for local smtp mocking etc
	envSMTPNoTLS = "RM_SMTP_NOTLS"
	// use starttls when notls was used
	envSMTPStartTLS = "RM_SMTP_STARTTLS"
	// envSMTPInsecureTLS dont check cert (bad)
	envSMTPInsecureTLS = "RM_SMTP_INSECURE_TLS"
	// envSMTPFrom custom from address
	envSMTPFrom = "RM_SMTP_FROM"

	// envHwrApplicationKey the myScript application key
	envHwrApplicationKey = "RMAPI_HWR_APPLICATIONKEY"
	// envHwrHmac myScript hmac key
	envHwrHmac = "RMAPI_HWR_HMAC"
	// EnvLogFile log file to use
	EnvLogFile     = "RM_LOGFILE"
	envHTTPSCookie = "RM_HTTPS_COOKIE"
	envTrustProxy  = "RM_TRUST_PROXY"
)

// Config config
type Config struct {
	Port              string
	StorageURL        string
	DataDir           string
	RegistrationOpen  bool
	CreateFirstUser   bool
	JWTSecretKey      []byte
	JWTRandom         bool
	Certificate       tls.Certificate
	SMTPConfig        *email.SMTPConfig
	LogFile           string
	HWRApplicationKey string
	HWRHmac           string
	HTTPSCookie       bool
	TrustProxy        bool
}

// Verify verify
func (cfg *Config) Verify() {
	if cfg.JWTRandom {
		log.Warn("The authentication will fail the next time you start the server!")
		log.Warnf("%s was not set! The following was autogenerated", envJWTSecretKey)
		log.Warnf("%s=%X", envJWTSecretKey, cfg.JWTSecretKey)
	}

	if !cfg.HTTPSCookie {
		log.Warnln(envHTTPSCookie + " is not set, use only when not using https!")
	}

	if cfg.SMTPConfig == nil {
		log.Warnln("smtp not configured, no emails will be sent")
	}

	if cfg.HWRApplicationKey == "" {
		log.Info("if you want HWR, provide the myScript applicationKey in: " + envHwrApplicationKey)
	}
	if cfg.HWRHmac == "" {
		log.Info("provide the myScript hmac in: " + envHwrHmac)
	}
}

// FromEnv config from environment values
func FromEnv() *Config {
	var err error
	var dataDir string
	data := os.Getenv(envDataDir)
	if data != "" {
		dataDir = data
	} else {
		dataDir, err = filepath.Abs(DefaultDataDir)
		if err != nil {
			log.Fatal("DataDir: ", err)
		}
	}

	port := os.Getenv(envPort)
	if port == "" {
		port = DefaultPort
	}

	jwtGenerated := false
	jwtSecretKey := []byte(os.Getenv(envJWTSecretKey))
	if len(jwtSecretKey) == 0 {
		jwtSecretKey = make([]byte, 32)
		_, err := rand.Read(jwtSecretKey)
		if err != nil {
			log.Fatal(err)
		}
		jwtGenerated = true
	}
	dk := pbkdf2.Key(jwtSecretKey, []byte("todo some salt"), 10000, 32, sha256.New)

	var cert tls.Certificate
	certPath := os.Getenv(envTLSCert)
	keyPath := os.Getenv(envTLSKey)
	if certPath != "" && keyPath != "" {

		cert, err = tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			log.Fatal("unable to load certificate:", err)
		}
	}
	openRegistration, _ := strconv.ParseBool(os.Getenv(envRegistrationOpen))
	httpsCookie, _ := strconv.ParseBool(os.Getenv(envHTTPSCookie))

	uploadURL := os.Getenv(EnvStorageURL)
	if uploadURL == "" {
		//it will go through the local proxy
		uploadURL = "https://" + DefaultHost
	}

	u, err := url.Parse(uploadURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == ""  {
		log.Fatalf("%s '%s' cannot be parsed, or missing scheme (http|https) %v", EnvStorageURL, uploadURL, err)
	}

	// smtp
	var smtpCfg *email.SMTPConfig
	servername := os.Getenv(envSMTPServer)

	if servername != "" {
		inSecureTLS, _ := strconv.ParseBool(os.Getenv(envSMTPInsecureTLS))
		noTLS, _ := strconv.ParseBool(os.Getenv(envSMTPNoTLS))
		startTLS, _ := strconv.ParseBool(os.Getenv(envSMTPStartTLS))
		smtpCfg = &email.SMTPConfig{
			Server:      servername,
			Username:    os.Getenv(envSMTPUsername),
			Password:    os.Getenv(envSMTPPassword),
			Helo:        os.Getenv(envSMTPHelo),
			NoTLS:       noTLS,
			StartTLS:    startTLS,
			InsecureTLS: inSecureTLS,
		}
		fromOverride := os.Getenv(envSMTPFrom)
		if fromOverride != "" {
			fromAddress, err := mail.ParseAddress(os.Getenv(envSMTPFrom))
			if err != nil {
				log.Warn(envSMTPFrom, " can't parse address: ", fromAddress, err)
			} else {
				smtpCfg.FromOverride = fromAddress
			}
		}
	}

	trustProxy, _ := strconv.ParseBool(os.Getenv(envTrustProxy))

	cfg := Config{
		Port:              port,
		StorageURL:        uploadURL,
		DataDir:           dataDir,
		JWTSecretKey:      dk,
		JWTRandom:         jwtGenerated,
		Certificate:       cert,
		RegistrationOpen:  openRegistration,
		SMTPConfig:        smtpCfg,
		HWRApplicationKey: os.Getenv(envHwrApplicationKey),
		HWRHmac:           os.Getenv(envHwrHmac),
		HTTPSCookie:       httpsCookie,
		TrustProxy:        trustProxy,
	}
	return &cfg
}

// EnvVars env vars usage
func EnvVars() string {
	return fmt.Sprintf(`
Environment Variables:

General:
	%s	Secret for signing JWT tokens
	%s	Url the tablet can resolve (default: %s)
			needs to be set to the hostname or proxy if behind a proxy
			especially if you want other tools to work (eg rmapi)

	%s	Log verbosity level (debug, info, warn) (default: info)
	%s	Log format: json (default: text)
	%s		Port (default: %s)
	%s		Local storage folder (default: %s)
	%s	Path to the server certificate.
	%s		Path to the server certificate key.
	%s	Write logs to file
	%s Send auth cookie only via https
	%s	Trust the proxy for X-Forwarded-For/X-Real-IP (set only if behind a proxy)

Emails, smtp:
	%s
	%s
	%s
	%s	no tls/plaintext, for testing or something
	%s	don't check the server certificate (not recommended)
	%s	custom HELO (if your email server needs it)
	%s	override the email's From:

myScript hwr (needs a developer account):
	%s
	%s
`,
		envJWTSecretKey,
		EnvStorageURL,
		DefaultHost,
		EnvLogLevel,
		EnvLogFormat,
		envPort,
		DefaultPort,
		envDataDir,
		DefaultDataDir,
		envTLSCert,
		envTLSKey,
		EnvLogFile,
		envHTTPSCookie,
		envTrustProxy,

		envSMTPServer,
		envSMTPUsername,
		envSMTPPassword,
		envSMTPNoTLS,
		envSMTPInsecureTLS,
		envSMTPHelo,
		envSMTPFrom,

		envHwrApplicationKey,
		envHwrHmac,
	)
}
