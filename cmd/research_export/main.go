// research_export reads an existing archive; it never migrates or writes it.
package main

import (
	"context"
	"flag"
	"fmt"
	"nofx/researchsnapshot"
	"os"
	"time"
)

func main() {
	db := flag.String("db", "data/data.db.research.db", "research archive path")
	from := flag.String("from", "", "inclusive RFC3339 receipt time")
	to := flag.String("to", "", "exclusive RFC3339 receipt time")
	flag.Parse()
	if err := run(*db, *from, *to); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(path, from, to string) error {
	lo, err := time.Parse(time.RFC3339, from)
	if err != nil {
		return err
	}
	hi, err := time.Parse(time.RFC3339, to)
	if err != nil {
		return err
	}
	a, err := researchsnapshot.OpenReadOnly(path)
	if err != nil {
		return err
	}
	defer a.Close()
	b, err := a.Export(context.Background(), lo.UnixMilli(), hi.UnixMilli())
	if err != nil {
		return err
	}
	if err = researchsnapshot.VerifyBundle(b); err != nil {
		return err
	}
	_, err = os.Stdout.Write(b)
	return err
}
