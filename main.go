package grmon

import (
	"bytes"
	"encoding/json"
	"net/http"
	"regexp"
	"runtime/pprof"
	"strconv"
	"strings"
)

var (
	newline   = byte(10)
	statusRe  = regexp.MustCompile("^goroutine\\s(\\d+)\\s\\[(.*)\\]:")
	createdRe = regexp.MustCompile("^created by (.*)")
	threadRe  = regexp.MustCompile("^threadcreate\\sprofile:\\stotal\\s(\\d+)")
)

type Routine struct {
	Num       int      `json:"no"`
	State     string   `json:"state"`
	CreatedBy string   `json:"created_by"`
	Trace     []string `json:"trace"`
}

func ReadRoutines() (routines []*Routine) {
	var p *Routine
	var buf bytes.Buffer

	pprof.Lookup("goroutine").WriteTo(&buf, 2)

	for {
		line, err := buf.ReadString(newline)
		if err != nil {
			break
		}

		mg := statusRe.FindStringSubmatch(line)
		if len(mg) > 2 {
			// new routine block
			p = &Routine{}

			i, err := strconv.Atoi(mg[1])
			if err != nil {
				panic(err)
			}
			p.Num = i

			p.State = mg[2]
			routines = append(routines, p)
			continue
		}

		mg = createdRe.FindStringSubmatch(line)
		if len(mg) > 1 {
			p.CreatedBy = mg[1]
		}

		line = strings.Trim(line, "\n")
		if line != "" {
			p.Trace = append(p.Trace, line)
		}
	}

	return
}

type ThreadCreate struct {
	Count int      `json:"count"`
	Trace []string `json:"trace"`
}

func ReadThreads() *ThreadCreate {
	var buf bytes.Buffer

	pprof.Lookup("threadcreate").WriteTo(&buf, 1)

	t := &ThreadCreate{}

	for {
		line, err := buf.ReadString(newline)
		if err != nil {
			break
		}

		mg := threadRe.FindStringSubmatch(line)
		if len(mg) > 1 {
			i, err := strconv.Atoi(mg[1])
			if err != nil {
				panic(err)
			}
			t.Count = i
			continue
		}

		line = strings.Trim(line, "\n")
		if line != "" {
			t.Trace = append(t.Trace, line)
		}
	}

	return t
}

func Start() { go http.ListenAndServe(":1234", nil) }

func grmonHandler(w http.ResponseWriter, r *http.Request) {
	routines := ReadRoutines()
	data, err := json.Marshal(routines)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(data)
}

func init() {
	http.HandleFunc("/debug/grmon", grmonHandler)
}
