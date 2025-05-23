package settings

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"golang.org/x/crypto/acme"
	"net"
	"ocelot/backend/certs"
	"ocelot/backend/tools"
	"strings"
	"time"
)

const (
	acmeAccountKeyField ConfigFieldKey = "ACME_ACCOUNT_KEY"

	productionCertEndpoint = "https://acme-v02.api.letsencrypt.org/directory"         // generates production cert
	selfSignedCertEndpoint = "https://acme-staging-v02.api.letsencrypt.org/directory" // generates self-signed cert for testing
)

var (
	sampleBaseKeyAuth     = "CaxCTxqOWo7FQjoRdRgxRoriPnOrp8PeMbCdPnh2Y84"
	sampleWildcardKeyAuth = "uJi0naVfLcobf2bK_t4-VkS0HFCK0U1WXkpGCGl3irE"
	acmeChallengePrefix   = "_acme-challenge."
)

type textDnsRecordInformation struct {
	Name            string `json:"host"`
	BaseKeyAuth     string `json:"base_key_auth"`
	WildcardKeyAuth string `json:"wildcard_key_auth"`
}

func crateCertificateViaLetsEncryptDns01Challenge(host, email string, challengeType tools.CertificateDnsChallengeClientType) (*textDnsRecordInformation, error) {
	var letsEncryptEndpointToAddress string

	if challengeType == tools.STUB_CERTIFICATE {
		return getDnsRecordStub(host), nil
	} else if challengeType == tools.FAKE_LETSENCRYPT_CERTIFICATE {
		letsEncryptEndpointToAddress = selfSignedCertEndpoint
	} else {
		letsEncryptEndpointToAddress = productionCertEndpoint
	}

	var data textDnsRecordInformation
	var err error
	data.Name = host
	client, certificateKey, err := generateKeysAndClient(email, letsEncryptEndpointToAddress)
	if err != nil {
		return nil, err
	}
	ctx, order, err := createOrder(host, client)
	if err != nil {
		return nil, err
	}
	baseChallenge, wildcardChallenge, baseAuthzURL, wildcardAuthzURL, err := fetchChallenges(ctx, client, order)
	if err != nil {
		return nil, err
	}
	data.BaseKeyAuth, data.WildcardKeyAuth, err = generateChallengeRecords(client, baseChallenge, wildcardChallenge)
	if err != nil {
		return nil, err
	}

	go func() {
		err = waitForDnsEntryToBeReady(ctx, data.Name, data.BaseKeyAuth, data.WildcardKeyAuth)
		if err != nil {
			Logger.Info("DNS TXT record was not found")
		}
		err = acceptChallenges(ctx, client, baseChallenge, wildcardChallenge)
		if err != nil {
			Logger.Info("Failed to accept challenges: %v", err)
		}
		err = waitForValidation(ctx, client, *baseAuthzURL, *wildcardAuthzURL)
		if err != nil {
			Logger.Info("Failed to validate challenges: %v", err)
		}
		csr, err := generateAndValidateCSR(data.Name, certificateKey)
		if err != nil {
			Logger.Info("Failed to generate and validate CSR: %v", err)
		}
		err = finalizeAndSaveCertificate(ctx, client, order, csr, certificateKey)
		if err != nil {
			Logger.Info("Failed to finalize and save certificate: %v", err)
		}
	}()

	return &data, nil
}

func waitForDnsEntryToBeReady(ctx context.Context, host string, wantedDnsTextRecordValues ...string) error {
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{}
			return d.DialContext(ctx, "udp", "8.8.8.8:53")
		},
	}

	return Retry("Poll DNS for challenge TXT records", 1*time.Minute, 20*time.Minute, func() error {
		txt, err := r.LookupTXT(ctx, acmeChallengePrefix+host)
		if err != nil {
			Logger.Info("Failed to lookup TXT record")
			return err
		}
		foundAll := true
		for _, w := range wantedDnsTextRecordValues {
			present := false
			for _, t := range txt {
				if t == w || strings.Contains(t, w) {
					present = true
					break
				}
			}
			if !present {
				foundAll = false
				break
			}
		}
		if foundAll {
			return nil
		} else {
			return errors.New("DNS TXT record was found but values were not correct")
		}
	})
}

func generateKeysAndClient(email, letsEncryptEndpointToAddress string) (*acme.Client, *rsa.PrivateKey, error) {
	accountKey, err := loadOrCreateAccountKey()
	if err != nil {
		Logger.Error("Failed to get account key: %v", err)
		return nil, nil, errors.New("failed to get account key")
	}
	client := &acme.Client{
		Key:          accountKey,
		DirectoryURL: letsEncryptEndpointToAddress,
	}

	certificateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		Logger.Error("Failed to generate certificate key: %v", err)
		return nil, nil, errors.New("failed to generate certificate key")
	}

	ctx := context.Background()
	var contact []string
	if email != "" {
		contact = []string{"mailto:" + email}
	}
	account := &acme.Account{Contact: contact}
	if _, registrationError := client.Register(ctx, account, func(string) bool { return true }); registrationError != nil {
		if acmeError, ok := registrationError.(*acme.Error); !ok || acmeError.StatusCode != 409 {
			Logger.Error("Failed to register ACME account: %v", registrationError)
			return nil, nil, errors.New("failed to register ACME account")
		}
	}
	return client, certificateKey, nil
}

func loadOrCreateAccountKey() (*rsa.PrivateKey, error) {
	pemStr, err := ConfigsRepo.GetValue(acmeAccountKeyField)
	if err == nil && pemStr != "" {
		if blk, _ := pem.Decode([]byte(pemStr)); blk != nil {
			if k, err := x509.ParsePKCS1PrivateKey(blk.Bytes); err == nil {
				return k, nil
			}
		}
	}

	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)})
	pemString := string(pemBytes)
	_ = ConfigsRepo.SetConfigField(acmeAccountKeyField, pemString)
	return k, nil
}

func createOrder(host string, client *acme.Client) (context.Context, *acme.Order, error) {
	ctx := context.Background()
	order, err := client.AuthorizeOrder(ctx, []acme.AuthzID{
		{Type: "dns", Value: host},
		{Type: "dns", Value: "*." + host},
	})
	if err != nil {
		Logger.Error("Failed to authorize order: %v", err)
		return nil, nil, errors.New("failed to authorize order")
	}
	if len(order.AuthzURLs) < 2 {
		Logger.Error("Expected at least 2 authorizations for base and wildcard domains")
		return nil, nil, errors.New("expected at least 2 authorizations for base and wildcard domains")
	}
	return ctx, order, nil
}

func fetchChallenges(ctx context.Context, client *acme.Client, order *acme.Order) (*acme.Challenge, *acme.Challenge, *string, *string, error) {
	baseAuthzURL := order.AuthzURLs[0]
	baseAz, err := client.GetAuthorization(ctx, baseAuthzURL)
	if err != nil {
		Logger.Error("Failed to get base authorization: %v", err)
		return nil, nil, nil, nil, errors.New("failed to get base authorization")
	}
	var baseChallenge *acme.Challenge
	for _, ch := range baseAz.Challenges {
		if ch.Type == "dns-01" {
			baseChallenge = ch
			break
		}
	}
	wildcardAuthzURL := order.AuthzURLs[1]
	wildcardAz, err := client.GetAuthorization(ctx, wildcardAuthzURL)
	if err != nil {
		Logger.Error("Failed to get wildcard authorization: %v", err)
		return nil, nil, nil, nil, errors.New("failed to get wildcard authorization")
	}
	var wildcardChallenge *acme.Challenge
	for _, ch := range wildcardAz.Challenges {
		if ch.Type == "dns-01" {
			wildcardChallenge = ch
			break
		}
	}
	if baseChallenge == nil || wildcardChallenge == nil {
		Logger.Error("DNS-01 challenges not found for base or wildcard domains")
		return nil, nil, nil, nil, errors.New("DNS-01 challenges not found for base or wildcard domains")
	}
	return baseChallenge, wildcardChallenge, &baseAuthzURL, &wildcardAuthzURL, nil
}

func generateChallengeRecords(client *acme.Client, baseChallenge, wildcardChallenge *acme.Challenge) (string, string, error) {
	baseKeyAuth, err := client.DNS01ChallengeRecord(baseChallenge.Token)
	if err != nil {
		Logger.Error("Failed to generate DNS-01 challenge response for base domain: %v", err)
		return "", "", errors.New("failed to generate DNS-01 challenge response for base domain")
	}
	wildcardKeyAuth, err := client.DNS01ChallengeRecord(wildcardChallenge.Token)
	if err != nil {
		Logger.Error("Failed to generate DNS-01 challenge response for wildcard domain: %v", err)
		return "", "", errors.New("failed to generate DNS-01 challenge response for wildcard domain")
	}
	return baseKeyAuth, wildcardKeyAuth, nil
}

func acceptChallenges(ctx context.Context, client *acme.Client, baseChallenge, wildcardChallenge *acme.Challenge) error {
	_, err := client.Accept(ctx, baseChallenge)
	if err != nil {
		Logger.Error("failed to accept base domain challenge: %v", err)
		return errors.New("failed to accept base domain challenge")
	}
	_, err = client.Accept(ctx, wildcardChallenge)
	if err != nil {
		Logger.Error("failed to accept wildcard domain challenge: %v", err)
		return errors.New("failed to accept wildcard domain challenge")
	}
	return nil
}

func waitForValidation(ctx context.Context, client *acme.Client, baseAuthzURL, wildcardAuthzURL string) error {
	return Retry("Poll Let's Encrypt for DNS challenge validation", 1*time.Minute, 5*time.Minute, func() error {
		baseAz, err := client.GetAuthorization(ctx, baseAuthzURL)
		if err != nil {
			Logger.Error("Failed to get base domain authorization: %v", err)
			return errors.New("failed to get base domain authorization")
		}
		wildcardAz, err := client.GetAuthorization(ctx, wildcardAuthzURL)
		if err != nil {
			Logger.Error("Failed to get wildcard domain authorization: %v", err)
			return errors.New("failed to get wildcard domain authorization")
		}
		if baseAz.Status == acme.StatusInvalid || wildcardAz.Status == acme.StatusInvalid {
			Logger.Error("Challenge validation failed")
			return errors.New("challenge validation failed")
		}
		if baseAz.Status == acme.StatusValid && wildcardAz.Status == acme.StatusValid {
			return nil
		} else {
			return errors.New("challenge validation not yet complete")
		}
	})
}

func generateAndValidateCSR(host string, certificateKey *rsa.PrivateKey) ([]byte, error) {
	csr := &x509.CertificateRequest{Subject: pkix.Name{CommonName: host}, DNSNames: []string{host, "*." + host}}
	data, err := x509.CreateCertificateRequest(rand.Reader, csr, certificateKey)
	if err != nil {
		Logger.Error("failed to generate CSR: %v", err)
		return nil, errors.New("failed to generate CSR")
	}
	return data, nil
}

func finalizeAndSaveCertificate(ctx context.Context, client *acme.Client, order *acme.Order, csr []byte, certificateKey *rsa.PrivateKey) error {
	var cert [][]byte
	err := Retry("Finalize certificate issuance with Let's Encrypt", 1*time.Minute, 10*time.Minute, func() error {
		var err error
		cert, _, err = client.CreateOrderCert(ctx, order.FinalizeURL, csr, true)
		if err != nil {
			Logger.Info("failed to finalize order: %v", err)
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	var certPEM, keyPEM []byte
	for _, c := range cert {
		certPEM = append(certPEM, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: c})...)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(certificateKey)})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		Logger.Error("failed to parse generated cert: %v", err)
		return errors.New("failed to parse generated cert")
	}
	if err = certs.SaveCert(&tlsCert); err != nil {
		Logger.Error("failed to save generated cert: %v", err)
		return errors.New("failed to save generated cert")
	}
	Logger.Info("new certificate was saved and loaded successfully")
	return nil
}

func Retry(operationName string, frequency, maxTime time.Duration, fn func() error) error {
	deadline := time.Now().Add(maxTime)
	for {
		Logger.Info("Retrying operation: %s", operationName)
		if err := fn(); err == nil {
			Logger.Info("Retry successful of operation: %s", operationName)
			return nil
		} else if time.Now().After(deadline) {
			Logger.Error("Retry deadline exceeded for operation: %s", operationName)
			return errors.New("retry deadline exceeded for operation: " + operationName)
		} else {
			Logger.Info("Attempt failed for operation '%s': %v, waiting...", operationName, err)
			time.Sleep(frequency)
		}
	}
}

func getDnsRecordStub(host string) *textDnsRecordInformation {
	txtDnsRecordToCreate := &textDnsRecordInformation{
		Name:            acmeChallengePrefix + host,
		BaseKeyAuth:     sampleBaseKeyAuth,
		WildcardKeyAuth: sampleWildcardKeyAuth,
	}
	return txtDnsRecordToCreate
}
