package services_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
)

func TestSurveyDefinitionV1(t *testing.T) {
	def := services.CurrentSurveyDefinition()
	if def.Version != "v1" || len(def.Questions) != 10 {
		t.Fatalf("definition = %s with %d questions", def.Version, len(def.Questions))
	}
	for i, q := range def.Questions {
		if want := fmt.Sprintf("q%02d", i+1); q.Code != want {
			t.Errorf("question %d code = %s, want %s", i, q.Code, want)
		}
	}
}

func TestParseSurveyDefinitionRejectsBrokenDefinitions(t *testing.T) {
	for name, js := range map[string]string{
		"no questions":   `{"version":"v9","questions":[]}`,
		"duplicate code": `{"version":"v9","questions":[{"code":"q01","type":"rating_1_5"},{"code":"q01","type":"rating_1_5"}]}`,
		"unknown type":   `{"version":"v9","questions":[{"code":"q01","type":"slider"}]}`,
		"no options":     `{"version":"v9","questions":[{"code":"q01","type":"single_choice"}]}`,
		"dup options":    `{"version":"v9","questions":[{"code":"q01","type":"multi_choice","options":["a","a"]}]}`,
		"no maxLength":   `{"version":"v9","questions":[{"code":"q01","type":"text"}]}`,
	} {
		if _, err := services.ParseSurveyDefinition([]byte(js)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// decodeAnswers mimics the JSON decoding of the request body.
func decodeAnswers(t *testing.T, js string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(js), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestValidateSurveyAnswers(t *testing.T) {
	def := services.CurrentSurveyDefinition()

	got, err := services.ValidateSurveyAnswers(&def, decodeAnswers(t,
		`{"q01":5,"q02":null,"q05":"business","q08":["quiet","location"],"q09":"yes","q10":"  Lovely stay.  "}`))
	if err != nil {
		t.Fatalf("valid answers rejected: %v", err)
	}
	js, _ := json.Marshal(got)
	if string(js) != `{"q01":5,"q05":"business","q08":["quiet","location"],"q09":"yes","q10":"Lovely stay."}` {
		t.Fatalf("normalized = %s", js)
	}

	// Empty optional answers are dropped, not rejected.
	got, err = services.ValidateSurveyAnswers(&def, decodeAnswers(t, `{"q01":3,"q08":[],"q10":"   "}`))
	if err != nil || len(got) != 1 {
		t.Fatalf("empty optional answers: %v %v", got, err)
	}

	long := strings.Repeat("あ", 1001)
	for name, tc := range map[string]struct{ answers, loc string }{
		"unknown question":     {`{"q01":4,"q99":"x"}`, "body.answers"},
		"rating too high":      {`{"q01":6}`, "body.answers.q01"},
		"rating zero":          {`{"q01":0}`, "body.answers.q01"},
		"rating fractional":    {`{"q01":4.5}`, "body.answers.q01"},
		"rating as string":     {`{"q01":"5"}`, "body.answers.q01"},
		"invalid option":       {`{"q01":4,"q05":"space_travel"}`, "body.answers.q05"},
		"multi not array":      {`{"q01":4,"q08":"quiet"}`, "body.answers.q08"},
		"multi invalid option": {`{"q01":4,"q08":["quiet","pool"]}`, "body.answers.q08"},
		"multi duplicate":      {`{"q01":4,"q08":["quiet","quiet"]}`, "body.answers.q08"},
		"text too long":        {`{"q01":4,"q10":"` + long + `"}`, "body.answers.q10"},
		"text not string":      {`{"q01":4,"q10":42}`, "body.answers.q10"},
		"required missing":     {`{"q05":"business"}`, "body.answers.q01"},
	} {
		_, err := services.ValidateSurveyAnswers(&def, decodeAnswers(t, tc.answers))
		if !errors.Is(err, services.ErrValidation) {
			t.Errorf("%s: err = %v, want ErrValidation", name, err)
			continue
		}
		fields := services.FieldsOf(err)
		if len(fields) == 0 || fields[0].Location != tc.loc {
			t.Errorf("%s: fields = %v, want location %s", name, fields, tc.loc)
		}
		for _, f := range fields {
			if strings.Contains(f.Location+f.Message, "q99") || strings.Contains(f.Message, "space_travel") {
				t.Errorf("%s: error echoes input: %v", name, f)
			}
		}
	}
	// Exactly 1000 characters is allowed.
	if _, err := services.ValidateSurveyAnswers(&def, decodeAnswers(t, `{"q01":4,"q10":"`+strings.Repeat("あ", 1000)+`"}`)); err != nil {
		t.Fatalf("1000 characters rejected: %v", err)
	}
}

func TestSummarizeSurveySuppressesSmallCellsAndNeverReturnsText(t *testing.T) {
	def := services.CurrentSurveyDefinition()
	secret := "SECRET-FREE-TEXT"
	var rows []string
	for i := range 5 {
		rows = append(rows, fmt.Sprintf(`{"q01":%d,"q05":"business","q08":["quiet","location"],"q10":"%s %d"}`, i%5+1, secret, i))
	}
	rows = append(rows, `{"q01":5,"q02":4}`, `not json`, `{"q01":9,"q05":"bogus"}`)

	sum := services.SummarizeSurvey(&def, rows, 5)
	js, _ := json.Marshal(sum)
	if strings.Contains(string(js), secret) {
		t.Fatal("summary contains free text")
	}
	byCode := map[string]int{}
	for i, q := range sum {
		byCode[q.Code] = i
	}

	q01 := sum[byCode["q01"]]
	if q01.Suppressed || q01.Answered != 6 || q01.Mean == nil || *q01.Mean != 3.33 {
		t.Fatalf("q01 = %+v", q01)
	}
	if len(q01.Counts) != 5 || q01.Counts[4].Value != "5" || q01.Counts[4].Count != 2 {
		t.Fatalf("q01 counts = %+v", q01.Counts)
	}
	// q02 answered once: suppressed, no cells.
	if q02 := sum[byCode["q02"]]; !q02.Suppressed || q02.Answered != 1 || q02.Mean != nil || q02.Counts != nil {
		t.Fatalf("q02 = %+v", q02)
	}
	q05 := sum[byCode["q05"]]
	if q05.Suppressed || q05.Answered != 5 || q05.Counts[1].Value != "business" || q05.Counts[1].Count != 5 {
		t.Fatalf("q05 = %+v", q05)
	}
	if q08 := sum[byCode["q08"]]; q08.Answered != 5 || q08.Counts[1].Value != "quiet" || q08.Counts[1].Count != 5 {
		t.Fatalf("q08 = %+v", q08)
	}
	if q10 := sum[byCode["q10"]]; q10.Answered != 5 || q10.Suppressed || q10.Counts != nil || q10.Mean != nil {
		t.Fatalf("q10 = %+v", q10)
	}

	// Below the threshold everything is suppressed.
	for _, q := range services.SummarizeSurvey(&def, rows[:4], 5) {
		if !q.Suppressed || q.Counts != nil || q.Mean != nil {
			t.Fatalf("%s not suppressed with 4 responses: %+v", q.Code, q)
		}
	}
}
