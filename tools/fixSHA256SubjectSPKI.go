package main

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/mozilla/tls-observatory/certificate"
	"github.com/mozilla/tls-observatory/database"
)

func main() {
	db, err := database.RegisterConnection(
		os.Getenv("TLSOBS_POSTGRESDB"),
		os.Getenv("TLSOBS_POSTGRESUSER"),
		os.Getenv("TLSOBS_POSTGRESPASS"),
		os.Getenv("TLSOBS_POSTGRES"),
		"require")
	defer db.Close()
	if err != nil {
		panic(err)
	}
	offset := 0
	limit := 100
	if len(os.Args) > 1 {
		offset, err = strconv.Atoi(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
	}
	for {
		fmt.Printf("\nProcessing offset %d to %d: ", offset, offset+limit)
		rows, err := db.Query(`SELECT id, raw_cert
					FROM certificates
					WHERE id > $1
					ORDER BY id ASC LIMIT $2`, offset, limit)
		if rows != nil {
			defer rows.Close()
		}
		if err != nil {
			panic(fmt.Errorf("Error while retrieving certs: '%v'", err))
		}
		i := 0
		for rows.Next() {
			i++
			var raw string
			var id int64
			err = rows.Scan(&id, &raw)
			if err != nil {
				fmt.Println("error while parsing cert", id, ":", err)
				continue
			}
			certdata, err := base64.StdEncoding.DecodeString(raw)
			if err != nil {
				fmt.Println("error decoding base64 of cert", id, ":", err)
				continue
			}
			c, err := x509.ParseCertificate(certdata)
			if err != nil {
				fmt.Println("error while x509 parsing cert", id, ":", err)
				continue
			}
			_, err = db.Exec(`UPDATE certificates SET sha256_subject_spki=$1 WHERE id=$2`,
				certificate.SHA256SubjectSPKI(c), id)
			if err != nil {
				fmt.Println("error while updating cert", id, "in database:", err)
			}
		}
		if i == 0 {
			fmt.Println("done!")
			break
		}
		//offset += limit
		offset += limit
	}
}
