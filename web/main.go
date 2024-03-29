package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	rethink "github.com/dancannon/gorethink"
)

const (
	dbName    = "dockerdemo"
	tblVisits = "visits"
)

var (
	dbAddr   string
	session  *rethink.Session
	hostname string
)

type Visit struct {
	Server  string
	Visitor string
}

func initdb() {
	// retries
	for i := 0; i < 5; i++ {
		s, err := rethink.Connect(rethink.ConnectOpts{
			Address:  dbAddr,
			Database: dbName,
		})
		if err != nil {
			log.Printf("unable to connect; retrying: %s", err)
			time.Sleep(2 * time.Second)
			continue
		}

		session = s
	}

	if session == nil {
		log.Fatalf("unable to get database connection")
	}

	// Create database
	rethink.DBCreate(dbName).Run(session)

	// Check if table exists
	_, err := rethink.Table(tblVisits).Run(session)
	if err != nil {
		// If not, create it
		if _, err = rethink.DB(dbName).TableCreate(tblVisits).Run(session); err != nil {
			log.Fatalf("error creating table: %s", err)
		}
	}
}

func recordVisit(r *http.Request) error {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)

	var visit = Visit{
		Server:  hostname,
		Visitor: ip,
	}

	_, err := rethink.Table(tblVisits).Insert(visit).RunWrite(session)
	return err
}

func fetchAllVisits() ([]Visit, error) {
	rows, err := rethink.Table(tblVisits).Run(session)
	if err != nil {
		return nil, err
	}

	var visits []Visit
	err = rows.All(&visits)
	if err != nil {
		return nil, err
	}

	return visits, nil
}

func handleError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), 500)
}

func handleVisitor(w http.ResponseWriter, r *http.Request) {
	var err error
	if err = recordVisit(r); err != nil {
		handleError(w, err)
	}

	visits, err := fetchAllVisits()
	if err != nil {
		handleError(w, err)
	}

	var responsebuf bytes.Buffer
	fmt.Fprintf(&responsebuf, "You are visiting container: %s\n\n", hostname)

	// Get recent visits in reverse order (newest at the top)
	fmt.Fprintf(&responsebuf, "Recent Visits:\n")
	last := len(visits) - 1
	for i := range visits {
		fmt.Fprintf(&responsebuf, "%s\t%s\n", visits[last-i].Server, visits[last-i].Visitor)
	}

	fmt.Fprintf(w, responsebuf.String())
}

func main() {
	flag.StringVar(&dbAddr, "addr", "db:28015", "Address of rethinkdb (e.g. db:28015)")
	flag.Parse()

	var err error
	hostname, err = os.Hostname()
	if err != nil {
		log.Fatalf("Could not get hostname: %s", err)
	}

	initdb()

	http.HandleFunc("/", handleVisitor)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
