package cluster

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	APIServerURL       = "https://kubernetes.default:443"
	K8sAPIServerCertCN = "apiserver"
	K3sAPIServerCertCN = "k3s"
)

func GetServerCertExpiredTime(serverURL string) (*time.Time, error) {
	conf := &tls.Config{
		InsecureSkipVerify: true,
	}

	u, err := url.Parse(APIServerURL)
	if err != nil {
		return nil, err
	}
	conn, err := tls.Dial("tcp", u.Host, conf)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	invalidConns := []string{}
	for _, cert := range conn.ConnectionState().PeerCertificates {
		if strings.Contains(cert.Subject.CommonName, K8sAPIServerCertCN) ||
			strings.Contains(cert.Subject.CommonName, K3sAPIServerCertCN) {
			return &cert.NotAfter, nil
		}
		invalidConns = append(invalidConns, cert.Subject.CommonName)
	}

	return nil, fmt.Errorf("cert CN not contains apiserver: %s", strings.Join(invalidConns, ","))
}
