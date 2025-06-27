package main

import (
	"crypto/x509/pkix"
	"fmt"
	"log"

	"github.com/go-pantheon/fabrica-util/security/certificate"
)

func main() {
	cert, err := certificate.CreateSelfSignedCert(pkix.Name{
		CommonName: "janus.go-pantheon.dev",
		Country:    []string{"SG"},
		Province:   []string{"Singapore"},
		Locality:   []string{"Singapore"},
		Organization: []string{
			"Pantheon",
			"Janus",
		},
	}, 365)
	if err != nil {
		log.Fatal(err)
	}

	pri, err := certificate.ExportPriToPEM(cert.KeyPair.Pri)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nprivate PEM: \n%s\n", string(pri))
	fmt.Printf("\ncert PEM: \n%s\n", string(cert.CertPEM))

	pub, err := certificate.ExportPubToPEM(cert.KeyPair.Pub)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\npublic PEM: \n%s\n", string(pub))

	fmt.Printf("subject: %s\n", cert.X509Cert.Subject.String())
	fmt.Printf("issuer: %s\n", cert.X509Cert.Issuer.String())
	fmt.Printf("not before: %s\n", cert.X509Cert.NotBefore.String())
	fmt.Printf("not after: %s\n", cert.X509Cert.NotAfter.String())
	fmt.Printf("serial: %s\n", cert.X509Cert.SerialNumber.String())

	org := []byte("hello world")
	signRet, err := certificate.Sign(cert.KeyPair.Pri, org)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("signature: %s\n", certificate.EncodeBase64(signRet.Sign))

	valid := certificate.Verify(cert.KeyPair.Pub, org, signRet.Sign)
	fmt.Printf("signature verification result: %t\n", valid)

	fmt.Println("succeed")
}
