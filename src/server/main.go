package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"github.com/manticoresoftware/go-sdk/manticore"

	"mc_fts/src/server/sqlstore"
	"mc_fts/src/server/templates"
)

var db *sql.DB

type result struct {
	IncidentId  string
	DisplayName string
	Description string
	Snippet     template.HTML
}

type homeItem struct {
	IncidentId string
	RuleId     string
	Target     string
	Snippet    template.HTML
}

type searchItem struct {
	IncidentId string
	Fields     string
	Snippet    template.HTML
}

type host struct {
	HostId      string
	DisplayName string
}

type link struct {
	LinkId      string
	DisplayName string
}

type incidentFields struct {
	IncidentId string
	Fields     string
}

type incidentFull struct {
	IncidentId string
	Rule       string
	Host       *string
	Link       *string
	Snippet    template.HTML
}

type Anal struct{}

func (a Anal) Print(searchTimes []float32) {
	fmt.Printf("Median (%v): %v %v \n", len(searchTimes), a.CalcMedian(searchTimes), "ms")
	fmt.Printf("Avg (%v)   : %v %v \n", len(searchTimes), a.CalcAvg(searchTimes), "ms")
	fmt.Printf("Min (%v)   : %v %v \n", len(searchTimes), a.CalcMin(searchTimes), "ms")
	fmt.Printf("Max (%v)   : %v %v \n", len(searchTimes), a.CalcMax(searchTimes), "ms")
}

func (a Anal) CalcMedian(n []float32) float32 {
	sort.Slice(n, func(i, j int) bool { return n[i] < n[j] })
	mNumber := len(n) / 2

	if len(n)%2 != 0 {
		return n[mNumber]
	}

	return (n[mNumber-1] + n[mNumber]) / 2
}

func (a Anal) CalcAvg(n []float32) float32 {
	var sum float32 = 0

	for _, t := range n {
		sum += t
	}

	return sum / float32(len(n))
}

func (a Anal) CalcMin(n []float32) float32 {
	var min float32 = 1000

	for _, t := range n {
		if t < min {
			min = t
		}
	}

	return min
}

func (a Anal) CalcMax(n []float32) float32 {
	var max float32 = 0

	for _, t := range n {
		if t > max {
			max = t
		}
	}

	return max
}

func MCQuery(cl *manticore.Client, index string, query string) ([]string, *time.Duration, error) {
	mcQuery := strings.Join(strings.Split(query, " "), "|")
	search := manticore.NewSearch(mcQuery, index, "")
	search.Limit = 100
	search.MaxMatches = 50000
	qres, err := cl.RunQuery(search)
	if err != nil {
		return nil, nil, err
	}

	results := make([]string, 0)

	for _, match := range qres.Matches {
		results = append(results, match.Attrs[0].(manticore.JsonOrStr).String())
	}

	return results, &qres.QueryTime, nil
}

func SearchIncidents(cl *manticore.Client, query string) ([]incidentFull, *time.Duration, error) {
	incidentIDs, qtime, err := MCQuery(cl, "incidents", query)
	if err != nil {
		return nil, nil, err
	}

	start := time.Now()

	incidentValues := make([]string, 0)
	for i, ruleID := range incidentIDs {
		incidentValues = append(incidentValues, fmt.Sprintf("(%v, '%v')", i, ruleID))
	}

	if len(incidentValues) == 0 {
		qt := time.Since(start) + *qtime
		return []incidentFull{}, &qt, nil
	}
	incidentValuesStr := strings.Join(incidentValues, ", \n")

	queryRaw := fmt.Sprintf(sqlstore.IncidentsList, incidentValuesStr)

	rows, err := db.Query(queryRaw)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	results := make([]incidentFull, 0)
	for rows.Next() {
		var r incidentFull
		var snip string
		if err := rows.Scan(&r.IncidentId); err != nil {
			return nil, nil, err
		}
		r.Snippet = template.HTML(strings.Replace(snip, "\n", "<br>", -1))
		results = append(results, r)
	}

	qt := time.Since(start) + *qtime

	return results, &qt, nil
}

func main() {
	words := []string{"layer", "opposite", "waist", "become", "address", "adult", "upper", "twelve", "card", "prefer", "patient", "concerning", "welcome", "bread", "connect", "beyond", "law", "northern", "more", "gray", "west", "except", "OK", "negative", "nation", "program", "plenty", "wine", "information", "produce", "animal", "smart", "fear", "lock", "upper", "physical", "beautiful", "truck", "steady", "card", "walk", "rock", "bear", "grass", "hand", "odd", "proof", "decrease", "represent", "over", "quiet", "solve", "require", "important", "inform", "nose", "very", "crowd", "third", "request", "woman", "practical", "invite", "adjective", "wake", "soon", "itself", "relation", "fork", "food", "average", "change", "well", "each", "quality", "supply", "point", "dollar", "child", "pound", "balance", "suddenly", "cook", "notice", "traffic", "recognize", "drunk", "toilet", "always", "say", "reason", "under", "forget", "replace", "medical", "clothes", "breast", "straight", "duck", "admit"}

	dburl := "postgresql://postgres@127.0.0.1:5432/nextdb?sslmode=disable"

	var err error
	if db, err = sql.Open("postgres", dburl); err != nil {
		log.Fatal(err)
	}

	cl := manticore.NewClient()
	cl.SetServer("127.0.0.1", 9313)
	cl.Open()

	//rows, err := db.Query(sqlstore.IncidentsListFull)
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//defer rows.Close()
	//results := make([]incidentFull, 0, 10)
	//for rows.Next() {
	//	var h incidentFull
	//	if err := rows.Scan(&h.IncidentId, &h.Fields); err != nil {
	//		return
	//	}
	//	results = append(results, h)
	//}
	//
	//for i, h := range results {
	//	qStr := fmt.Sprintf(
	//		`insert into incidents values(%v, '%v', '%v')`,
	//		i,
	//		h.IncidentId,
	//		h.Fields,
	//	)
	//	_, err = cl.Sphinxql(qStr)
	//	if err != nil {
	//		fmt.Println(err.Error())
	//		break
	//	}
	//	fmt.Println(i+1, "/", len(results))
	//}

	tplHome := template.Must(template.New(".").Parse(templates.TplStrHome))
	tplResults := template.Must(template.New(".").Parse(templates.TplStrResults))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		q := r.FormValue("q")
		if q == "" {
			rows, err := db.Query(sqlstore.ListIncidentsFull)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			defer rows.Close()
			results := make([]homeItem, 0, 10)
			for rows.Next() {
				var r homeItem
				var snip string
				if err := rows.Scan(&r.IncidentId, &r.RuleId, &r.Target); err != nil {
					http.Error(w, err.Error(), 404)
					return
				}
				r.Snippet = template.HTML(strings.Replace(snip, "\n", "<br>", -1))
				results = append(results, r)
			}

			tplHome.Execute(w, map[string]interface{}{
				"Results": results,
			})
			return
		}
		if q == "calculate" {
			searchTimes := make([]float32, 0)
			for i := 0; i < 10; i++ {
				fmt.Println(i+1, "/", 10)
				for _, word := range words {
					_, qtime, err := SearchIncidents(&cl, word)
					if err != nil {
						http.Error(w, err.Error(), 500)
						return
					}
					if qtime.Milliseconds() > 0 {
						searchTimes = append(searchTimes, float32(qtime.Milliseconds()))
					}
				}
			}

			if len(searchTimes) == 0 {
				panic("Empty")
			}

			a := Anal{}
			a.Print(searchTimes)

			return
		}

		results, qtime, err := SearchIncidents(&cl, q)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		fmt.Println(qtime.Milliseconds())
		tplResults.Execute(w, map[string]interface{}{
			"Results": results,
			"Query":   q,
		})
	})

	const PORT = "4242"

	fmt.Println("Starting on", PORT, "PORT...")
	log.Fatal(http.ListenAndServe(":"+PORT, nil))
}
