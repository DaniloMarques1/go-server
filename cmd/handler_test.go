package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
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
	removeFile()
	os.Exit(code)
}

func createFile() error {
	json := []byte(`{ "person": [
		{"id": 1, "name": "Fitz", "age": 21},
		{"id": 2, "name": "Batman", "age": 25},
		{"id": 3, "name": "Joao", "age": 32},
		{"id": 4, "name": "Zé", "age": 21},
		{"id": 5, "name": "Beeh", "age": 23},
		{"id": 6, "name": "Foo", "age": 22},
		{"id": 7, "name": "John", "age": 18},
		{"id": 8, "name": "Zac", "age": 30},
		{"id": 9, "name": "Dee", "age": 45},
		{"id": 10, "name": "Mike", "age": 46},
		{"id": 11, "name": "Nikao", "age": 21},
		{"id": 12, "name": "Nilo", "age": 31},
		{"id": 13, "name": "Jade", "age": 37},
		{"id": 14, "name": "Jack", "age": 15},
		{"id": 15, "name": "Eminem", "age": 35},
		{"id": 16, "name": "Snoop", "age": 50},
		{"id": 17, "name": "Dre", "age": 35},
		{"id": 18, "name": "Lewa", "age": 33}
	]}`)

	if err := os.WriteFile(TEMPFILE, json, 0777); err != nil {
		return err
	}
	return nil
}

func resetDB() error {
	db, err := handler.readDB()
	if err != nil {
		return err
	}
	handler.db = db
	return nil
}

func removeFile() error {
	if err := os.Remove(TEMPFILE); err != nil {
		return err
	}
	return nil
}

func executeRequest(request *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	handler.router.ServeHTTP(rr, request)
	return rr
}

func parseResponse(resp []byte, res interface{}) error {
	if err := json.Unmarshal(resp, res); err != nil {
		return err
	}

	return nil
}

func TestSave(t *testing.T) {
	createFile()
	resetDB()
	cases := []struct {
		label              string
		body               string
		expectedStatusCode int
		last               bool
	}{
		{"Should return http 201", `{"id": 1, "name":"Michael", "age": 40}`, http.StatusCreated, false},
		{"Should return http 400", `wrong body type`, http.StatusBadRequest, false},
		{"Should return http 500", `{"id": 1, "name":"Michael", "age": 40}`, http.StatusInternalServerError, true},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			if tc.last {
				// so that writeDB returns an error
				handler.fileName = ""
			}
			req := httptest.NewRequest(http.MethodPost, "/person", strings.NewReader(tc.body))
			response := executeRequest(req)
			if response.Code != tc.expectedStatusCode {
				t.Fatalf("Wrong status code. Expect: %v got: %v\n", tc.expectedStatusCode, response.Code)
			}
			if tc.last {
				handler.fileName = TEMPFILE
			}
		})
	}
	removeFile()
}

func TestFindAll(t *testing.T) {
	createFile()
	resetDB()
	cases := []struct {
		label          string
		page           string
		pageSize       string
		lenghtReturned int
		status         int
	}{
		{"Should return http 200", "0", fmt.Sprintf("%v", len(handler.db["person"])), len(handler.db["person"]), http.StatusOK},
		{"Should return http 200 with pagination page 0", "0", "3", 3, http.StatusOK},
		{"Should return http 200 with pagination page 1", "1", "3", 3, http.StatusOK},
		{"Should return http 200 with pagination page 1", "1", "5", 5, http.StatusOK},
		{"Should return http 200 with pagination page 1", "3", "5", 3, http.StatusOK},
		{"Should return http 400 with page invalid", "invalid", "5", 3, http.StatusBadRequest},
		{"Should return http 400 with page_size invalid", "0", "invalid", 3, http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/person?page=%v&page_size=%v", tc.page, tc.pageSize), nil)
			response := executeRequest(req)
			if response.Code != tc.status {
				t.Fatalf("Wrong status code. Expected: %v got: %v\n", tc.status, response.Code)
			}

			if response.Code != http.StatusBadRequest {
				res := make(DatabaseType)
				err := parseResponse(response.Body.Bytes(), &res)
				if err != nil {
					t.Fatal(err)
				}

				if len(res["person"]) != tc.lenghtReturned {
					t.Fatalf("Should have returned lenght %v, instead returned length %v\n",
					len(handler.db), len(res["person"]))
				}
			}
		})
	}

	removeFile()
}

func TestFindById(t *testing.T) {
	createFile()
	resetDB()

	cases := []struct {
		label          string
		id             string
		expected       string
		expectedStatus int
		errReturned    bool
	}{
		{"Search for id 2", "2", "Batman", http.StatusOK, false},
		{"Search for id 1", "1", "Fitz", http.StatusOK, false},
		{"Search for id that does not exist", "100", ElementNotFound, http.StatusNotFound, true},
		{"Search for invalid id", "Invalid", InvalidId, http.StatusBadRequest, true},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/person/%v", tc.id), nil)
			response := executeRequest(req)
			if response.Code != tc.expectedStatus {
				t.Fatalf("Wrong status code. Expected: %v got: %v\n", tc.expectedStatus, response.Code)
			}
			res := make(map[string]interface{})
			err := parseResponse(response.Body.Bytes(), &res)
			if err != nil {
				t.Fatal(err)
			}
			if tc.errReturned {
				if res["message"] != tc.expected {
					t.Fatalf("Expected element not found error\n")
				}
			} else {
				if res["name"] != tc.expected {
					t.Fatalf("Should have returned name: %v, instead got: %v\n", tc.expected, res["name"])
				}
			}
		})
	}

	removeFile()
}

func TestUpdateById(t *testing.T) {
	createFile()
	resetDB()

	cases := []struct {
		label          string
		id             string
		body           string
		updatedName    string
		expectedStatus int
		errReturned    bool
		errMsg         string
	}{
		{"Update for id 1", "1", `{"name": "FitzBoy"}`, "FitzBoy", http.StatusNoContent, false, ""},
		{"Update for id 2", "2", `{"name": "Robin"}`, "Robin", http.StatusNoContent, false, ""},
		{"Search for id that does not exist", "100", `{"name": "Robin"}`, "", http.StatusNotFound, true, ElementNotFound},
		{"Search for invalid id", "Invalid", `{"name": "Robin"}`, "", http.StatusBadRequest, true, InvalidId},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/person/%v", tc.id), strings.NewReader(tc.body))
			response := executeRequest(req)
			if response.Code != tc.expectedStatus {
				t.Fatalf("Wrong status code. Expected: %v got: %v\n", tc.expectedStatus, response.Code)
			}

			if tc.errReturned {
				res := make(map[string]interface{})
				err := parseResponse(response.Body.Bytes(), &res)
				if err != nil {
					t.Fatal(err)
				}
				if res["message"] != tc.errMsg {
					t.Fatalf("Expected element not found error\n")
				}
			} else {
				for _, item := range handler.db["person"] {
					id, _ := item["id"].(float64)
					tcId, _ := strconv.ParseFloat(tc.id, 64)
					if id == tcId {
						if item["name"] != tc.updatedName {
							t.Fatalf("Wrong updated name. Expect: %v got: %v\n", tc.updatedName, item["name"])
						}
					}
				}
			}
		})
	}

	removeFile()
}

func TestRemoveById(t *testing.T) {
	createFile()
	resetDB()

	cases := []struct {
		label          string
		id             string
		expectedStatus int
		errReturned    bool
		errMsg         string
	}{
		{"Search for id 2", "2", http.StatusNoContent, false, ""},
		{"Search for id 1", "1", http.StatusNoContent, false, ""},
		{"Search for id that does not exist", "100", http.StatusNotFound, true, ElementNotFound},
		{"Search for invalid id", "Invalid", http.StatusBadRequest, true, InvalidId},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/person/%v", tc.id), nil)
			response := executeRequest(req)
			if response.Code != tc.expectedStatus {
				t.Fatalf("Wrong status code. Expected: %v got: %v\n", tc.expectedStatus, response.Code)
			}

			if tc.errReturned {
				res := make(map[string]interface{})
				err := parseResponse(response.Body.Bytes(), &res)
				if err != nil {
					t.Fatal(err)
				}
				if res["message"] != tc.errMsg {
					t.Fatalf("Expected element not found error\n")
				}
			}
		})
	}

	removeFile()
}

func TestRemoveElement(t *testing.T) {
	cases := []struct {
		label          string
		slice          []map[string]interface{}
		idToRemove     float64
		lengthReturned int
		errReturned    string
	}{
		{"Should remove id 1", []map[string]interface{}{{"id": float64(1)}, {"id": float64(2)}}, 1, 1, ""},
		{"Should remove id 2", []map[string]interface{}{{"id": float64(1)}, {"id": float64(2)}}, 2, 1, ""},
		{"Should not remove element", []map[string]interface{}{{"id": float64(1)}, {"id": float64(2)}}, 3, 2, ElementNotFound},
		{"Should remove element", []map[string]interface{}{}, 1, 0, ElementNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			nSlice, err := removeElement(tc.slice, tc.idToRemove)
			if len(tc.errReturned) > 0 && err == nil {
				t.Fatal("Should have returned error")
			}
			if err != nil {
				if tc.errReturned != err.Error() {
					t.Fatalf("Wrong error returned. Expected: %v got: %v\n", tc.errReturned, err.Error())
				}
			} else {
				if len(nSlice) != tc.lengthReturned {
					t.Fatalf("Wrong length returned. Expected: %v got: %v\n", tc.lengthReturned, len(nSlice))
				}
			}
		})
	}
}
