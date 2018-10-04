package util

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	s "github.com/fullsailor/pkcs7"
)

var defaultDialer = &net.Dialer{Timeout: 16 * time.Second, KeepAlive: 16 * time.Second}

var cfg *tls.Config = &tls.Config{
	InsecureSkipVerify: true,
}
var client *http.Client = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig:     cfg,
		Dial:                defaultDialer.Dial,
		TLSHandshakeTimeout: 16 * time.Second,
	},
}

// DefaultHTTPClient retorna um cliente http configurado para dar um skip na validação do certificado digital
func DefaultHTTPClient() *http.Client {
	return client
}

//Post faz um requisição POST para uma URL e retorna o response, status e erro
func Post(url, body, timeout string, header map[string]string) (string, int, error) {
	return doRequest("POST", url, body, timeout, header)
}

//Get faz um requisição GET para uma URL e retorna o response, status e erro
func Get(url, body, timeout string, header map[string]string) (string, int, error) {
	return doRequest("GET", url, body, timeout, header)
}

func doRequest(method, url, body, timeout string, header map[string]string) (string, int, error) {
	client := DefaultHTTPClient()
	client.Timeout = GetDurationTimeoutRequest(timeout) * time.Second
	message := strings.NewReader(body)
	req, err := http.NewRequest(method, url, message)
	if err != nil {
		return "", http.StatusInternalServerError, err
	}
	if header != nil {
		for k, v := range header {
			req.Header.Add(k, v)
		}
	}
	resp, errResp := client.Do(req)
	if errResp != nil {
		return "", 0, errResp
	}
	defer resp.Body.Close()
	data, errResponse := ioutil.ReadAll(resp.Body)
	if errResponse != nil {
		return "", resp.StatusCode, errResponse
	}
	sData := string(data)
	return sData, resp.StatusCode, nil
}

//BuildTLSTransport creates a TLS Client Transport from crt, ca and key files
func BuildTLSTransport(crtPath string, keyPath string, caPath string) (*http.Transport, error) {
	cert, err := tls.LoadX509KeyPair(crtPath, keyPath)
	if err != nil {
		return nil, err
	}

	caCert, err := ioutil.ReadFile(caPath)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: true,
	}
	return &http.Transport{TLSClientConfig: tlsConfig}, nil
}

//Sigs request
func SignRequest(request string) (string, error) {

	pkey, err := parsePrivateKey()
	if err != nil {
		return "", err
	}

	chainCertificates, err := parseChainCertificates()
	if err != nil {
		return "", err
	}

	signedData, err := parseSignedData(request)
	if err != nil {
		return "", err
	}

	err = signedData.AddSigner(chainCertificates, pkey, s.SignerInfoConfig{})
	if err != nil {
		return "", err
	}

	detachedSignature, err := signedData.Finish()
	if err != nil {
		return "", err
	}

	signedRequest := base64.StdEncoding.EncodeToString(detachedSignature)

	return signedRequest, nil
}

func doRequestTLS(method, url, body, timeout string, header map[string]string, transport *http.Transport) (string, int, error) {
	var client *http.Client = &http.Client{
		Transport: transport,
	}
	client.Timeout = GetDurationTimeoutRequest(timeout) * time.Second
	b := strings.NewReader(body)
	req, err := http.NewRequest(method, url, b)
	if err != nil {
		return "", 0, err
	}

	if header != nil {
		for k, v := range header {
			req.Header.Add(k, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	// Dump response
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}
	sData := string(data)
	return sData, resp.StatusCode, nil
}

func PostTLS(url, body, timeout string, header map[string]string, transport *http.Transport) (string, int, error) {
	return doRequestTLS("POST", url, body, timeout, header, transport)
}

//HeaderToMap converte um http Header para um dicionário string -> string
func HeaderToMap(h http.Header) map[string]string {
	m := make(map[string]string)
	for k, v := range h {
		m[k] = v[0]
	}
	return m
}
