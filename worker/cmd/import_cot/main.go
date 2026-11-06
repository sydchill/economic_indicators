package main

import (
	"bufio"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/uptrace/bun/driver/pgdriver"
)

// Code-XXXXXX at end of the market header line. Codes can be alphanumeric
// and can end in + or Y, e.g. 12460+, 04360Y.
var codeRe = regexp.MustCompile(`Code-([A-Z0-9+]+)\s*$`)

// Report date line: "Commitments of Traders - Futures Only, October 06, 2026"
var dateRe = regexp.MustCompile(`Futures Only,\s+([A-Za-z]+\s+\d{1,2},\s+\d{4})`)

var dateFormats = []string{
	"January 02, 2006",
	"January 2, 2006",
}

// CFTC contract code → our ISO currency
var currencyByCode = map[string]string{
	"099741": "EUR", // EURO FX
	"096742": "GBP", // BRITISH POUND
	"097741": "JPY", // JAPANESE YEN
	"092741": "CHF", // SWISS FRANC
	"090741": "CAD", // CANADIAN DOLLAR
	"232741": "AUD", // AUSTRALIAN DOLLAR
	"112741": "NZD", // NZ DOLLAR
	"098662": "USD", // USD INDEX (ICE Futures U.S.)
}

var expectedCurrencies = []string{"EUR", "GBP", "JPY", "CHF", "CAD", "AUD", "NZD", "USD"}

type cotRecord struct {
	Currency string
	Date     time.Time
	OI       int64
	NCLong   int64
	NCShort  int64
}

type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

func main() {
	var files multiFlag
	flag.Var(&files, "file", "CFTC fixed-width file (repeatable)")
	apply := flag.Bool("apply", false, "write to DB")
	flag.Parse()

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "error: at least one --file is required")
		os.Exit(2)
	}

	var records []cotRecord
	for _, path := range files {
		recs, err := parseFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("%-60s %d record(s)\n", path, len(recs))
		records = append(records, recs...)
	}

	if len(records) == 0 {
		fmt.Fprintln(os.Stderr, "error: no matching FX contracts found in any file")
		os.Exit(1)
	}

	// Deduplicate on (currency, date) — later files override earlier
	dedup := map[string]cotRecord{}
	for _, r := range records {
		dedup[r.Currency+"|"+r.Date.Format("2006-01-02")] = r
	}
	records = records[:0]
	for _, r := range dedup {
		records = append(records, r)
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].Date.Equal(records[j].Date) {
			return records[i].Currency < records[j].Currency
		}
		return records[i].Date.Before(records[j].Date)
	})

	// Coverage report
	found := map[string]bool{}
	for _, r := range records {
		found[r.Currency] = true
	}
	var missing []string
	for _, c := range expectedCurrencies {
		if !found[c] {
			missing = append(missing, c)
		}
	}
	if len(missing) > 0 {
		fmt.Printf("WARN missing currencies: %s\n", strings.Join(missing, ", "))
	}
	fmt.Println()

	// Print table
	fmt.Printf("%-4s %-10s %-10s %-8s %-8s %s\n", "CUR", "DATE", "OI", "NC LONG", "NC SHORT", "NC NET")
	for _, r := range records {
		fmt.Printf("%-4s %-10s %-10d %-8d %-8d %+d\n",
			r.Currency, r.Date.Format("2006-01-02"), r.OI, r.NCLong, r.NCShort, r.NCLong-r.NCShort)
	}

	if !*apply {
		fmt.Println("\n(dry-run — pass --apply to write)")
		return
	}

	if err := writeDB(records); err != nil {
		fmt.Fprintln(os.Stderr, "db:", err)
		os.Exit(1)
	}
}

// parseFile walks the raw CFTC fixed-width file. The layout per market block:
//
//   MARKET NAME - EXCHANGE ... Code-XXXXXX
//   Commitments of Traders - Futures Only, <DATE>
//   <separator + headers, ~15 lines>
//   All  :   OI :  NC-L   NC-S  NC-Sp :  C-L    C-S   : TR-L  TR-S : NR-L   NR-S
//   Old  : ...
//   Other: ...
//   <rest of block, ignored>
//
// We use a small state machine per block:
//   - on "Code-XXXXXX":  if code is one we want, set currency and reset
//   - on date line:      capture report date
//   - on first "All  :": parse OI / NC long / NC short, emit record, reset
//
// This is why the state machine matters: "All  :" appears three times per
// block (data, changes, percent, traders), and we only want the first one.
func parseFile(path string) ([]cotRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var records []cotRecord
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)

	var currency string
	var date time.Time
	var dateFound bool

	for sc.Scan() {
		line := sc.Text()

		// 1. Market header with code
		if m := codeRe.FindStringSubmatch(line); m != nil {
			if cur, ok := currencyByCode[m[1]]; ok {
				currency = cur
				date = time.Time{}
				dateFound = false
			} else {
				currency = ""
				dateFound = false
			}
			continue
		}

		if currency == "" {
			continue
		}

		// 2. Report date
		if !dateFound {
			if m := dateRe.FindStringSubmatch(line); m != nil {
				if d, err := parseDate(m[1]); err == nil {
					date = d
					dateFound = true
				}
			}
			continue
		}

		// 3. First "All  :" after the date is the data row
		if strings.HasPrefix(strings.TrimSpace(line), "All  :") {
			rec, err := parseAllRow(line, currency, date)
			if err != nil {
				return nil, fmt.Errorf("%s %s: %w\n  line: %s",
					currency, date.Format("2006-01-02"), err, line)
			}
			records = append(records, rec)
			currency = ""
			date = time.Time{}
			dateFound = false
		}
	}
	return records, sc.Err()
}

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, f := range dateFormats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised date %q", s)
}

// parseAllRow splits on ':' since the fixed-width layout uses colons to
// separate the column groups. Four segments:
//
//	[0] "All  "
//	[1] OI
//	[2] NC long, NC short, NC spread, C long, C short, TR long, TR short (7 fields)
//	[3] NR long, NR short (2 fields)
func parseAllRow(line, currency string, date time.Time) (cotRecord, error) {
	parts := strings.Split(line, ":")
	if len(parts) < 4 {
		return cotRecord{}, fmt.Errorf("expected >=4 colon segments, got %d", len(parts))
	}

	oi, err := parseIntField(parts[1])
	if err != nil {
		return cotRecord{}, fmt.Errorf("OI: %w", err)
	}

	ncFields := strings.Fields(parts[2])
	if len(ncFields) < 2 {
		return cotRecord{}, fmt.Errorf("non-commercial segment has %d fields, want >=2", len(ncFields))
	}
	ncLong, err := parseIntField(ncFields[0])
	if err != nil {
		return cotRecord{}, fmt.Errorf("NC long: %w", err)
	}
	ncShort, err := parseIntField(ncFields[1])
	if err != nil {
		return cotRecord{}, fmt.Errorf("NC short: %w", err)
	}

	return cotRecord{
		Currency: currency,
		Date:     date,
		OI:       oi,
		NCLong:   ncLong,
		NCShort:  ncShort,
	}, nil
}

func parseIntField(s string) (int64, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	if s == "" || s == "." {
		return 0, fmt.Errorf("empty field")
	}
	return strconv.ParseInt(s, 10, 64)
}

func writeDB(records []cotRecord) error {
	_ = godotenv.Load()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL required")
	}
	db, err := sql.Open("pg", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	inserted, updated := 0, 0
	for _, r := range records {
		net := r.NCLong - r.NCShort
		var action string
		err := tx.QueryRowContext(ctx, `
			INSERT INTO market.cot_data
				(currency, report_date, noncommercial_long, noncommercial_short, noncommercial_net, source)
			VALUES ($1, $2, $3, $4, $5, 'cftc')
			ON CONFLICT (currency, report_date) DO UPDATE
			  SET noncommercial_long  = EXCLUDED.noncommercial_long,
			      noncommercial_short = EXCLUDED.noncommercial_short,
			      noncommercial_net   = EXCLUDED.noncommercial_net,
			      source              = EXCLUDED.source,
			      updated_at          = now()
			RETURNING CASE WHEN xmax = 0 THEN 'inserted' ELSE 'updated' END
		`, r.Currency, r.Date, r.NCLong, r.NCShort, net).Scan(&action)
		if err != nil {
			return fmt.Errorf("%s %s: %w", r.Currency, r.Date.Format("2006-01-02"), err)
		}
		if action == "inserted" {
			inserted++
		} else {
			updated++
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("\napplied: inserted=%d updated=%d\n", inserted, updated)
	return nil
}
