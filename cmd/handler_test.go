package cmd

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const TEMPFILE = "tmp"

var handler *Handler

func TestMain(m *testing.M) {
	if err := createFile(); err != nil {
		log.Fatal(err)
	}
	var err error
	handler, err = NewHandler(TEMPFILE, 8080)
	if err != nil {
		log.Fatal(err)
	}
	
	handler.RegisterRoutes("person")
	code := m.Run()
	os.Exit(code)
}

func createFile() error {
	json := []byte(`{"person": [{"name": "Fitz", "age": 21}, {"name": "Batman", "age": 25}]}`)
	if err := os.WriteFile(TEMPFILE, json, 0777); err != nil {
		return err
	}
	return nil
}

func executeRequest(request *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	handler.router.ServeHTTP(rr, request)
	return rr
}

func parseResponse(resp []byte) (DatabaseType, error) {
	res := make(DatabaseType)
	if err := json.Unmarshal(resp, &res); err != nil {
		return nil, err
	}

	return res, nil
}

func TestSave(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/person", nil)
	response := executeRequest(req)
	res, err := parseResponse(response.Body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(res["person"]) != 2 {
		t.Fatalf("Should have returned lenght 2, instead returned length %v\n", len(res["person"]))
	}
}
