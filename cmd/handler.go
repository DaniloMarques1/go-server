package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// messages constants
const (
	ElementNotFound = "Element Not found"
	InvalidId       = "Invalid ID"
	InvalidBody     = "Invalid Body"
	InvalidParams   = "Invalid parameters"
)

type ErrorDto struct {
	Message string `json:"message"`
}

// force each key to be an array of objects
type DatabaseType map[string][]map[string]interface{}

type Handler struct {
	db         DatabaseType
	router     *chi.Mux
	fileName   string
	serverPort int
	minified   bool
}

func NewHandler(fileName string, serverPort int, minified bool) (*Handler, error) {
	router := chi.NewRouter()
	router.Use(middleware)
	router.NotFound(handleNotFound)

	handler := &Handler{
		router:     router,
		fileName:   fileName,
		serverPort: serverPort,
	}

	db, err := handler.readDB()
	if err != nil {
		return nil, err
	}
	handler.db = db

	return handler, nil
}

// read the file given as argument and
// set as the handler database
func (h *Handler) readDB() (DatabaseType, error) {
	bytes, err := os.ReadFile(h.fileName)
	if err != nil {
		return nil, errors.New("Error reading the file. Make sure the file exists")
	}

	db := make(DatabaseType)
	err = json.Unmarshal(bytes, &db)
	if err != nil {
		return nil, errors.New("Error unmarshalling the json")
	}
	return db, nil
}

// write te current db state on file
func (h *Handler) writeDB() error {
	var bytes []byte
	var err error
	if h.minified {
		bytes, err = json.Marshal(h.db)
	} else {
		bytes, err = json.MarshalIndent(h.db, "", "  ")
	}
	if err != nil {
		return err
	}

	if err := os.WriteFile(h.fileName, bytes, 0777); err != nil {
		return err
	}
	return nil
}

// create REST endpoints for all the entities defined
// on the db.json file
func (h *Handler) RegisterRoutes(entity string) {
	h.router.Get(fmt.Sprintf("/%v", entity), func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%v %v\n", r.Method, r.URL.String())
		h.FindAll(entity, w, r)
		return
	})

	h.router.Get(fmt.Sprintf("/%v/{entityId}", entity), func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%v %v\n", r.Method, r.URL.String())
		h.FindById(entity, w, r)
		return
	})

	h.router.Post(fmt.Sprintf("/%v", entity), func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%v %v\n", r.Method, r.URL.String())
		h.Save(entity, w, r)
		return
	})

	h.router.Delete(fmt.Sprintf("/%v/{entityId}", entity), func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%v %v\n", r.Method, r.URL.String())
		h.RemoveById(entity, w, r)
		return
	})

	h.router.Put(fmt.Sprintf("/%v/{entityId}", entity), func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%v %v\n", r.Method, r.URL.String())
		h.Update(entity, w, r)
		return
	})
}

func (h *Handler) FindAll(entity string, w http.ResponseWriter, r *http.Request) {
	values := h.db[entity]
	q := r.URL.Query()
	page, err := strconv.Atoi(q.Get("page"))
	if err != nil {
		if len(q.Get("page")) > 0 {
			RespondERR(w, http.StatusBadRequest, InvalidParams)
			return
		}
		page = 0
	}
	pageSize, err := strconv.Atoi(q.Get("page_size"))
	if err != nil {
		if len(q.Get("page_size")) > 0 {
			RespondERR(w, http.StatusBadRequest, InvalidParams)
			return
		}
		pageSize = len(values)
	}

	skip := page

	end := pageSize * (page + 1)
	if end > len(values) {
		end = len(values)
	}
	if page > 0 {
		skip = page * pageSize
	}

	if skip > len(values) {
		json.NewEncoder(w).Encode(map[string]interface{}{entity: []interface{}{}})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{entity: values[skip:end]})
}

func (h *Handler) FindById(entity string, w http.ResponseWriter, r *http.Request) {
	slc := h.db[entity] // array of maps
	entityId, err := strconv.Atoi(chi.URLParam(r, "entityId"))
	if err != nil {
		RespondERR(w, http.StatusBadRequest, InvalidId)
		return
	}
	for _, item := range slc {
		itemId, ok := item["id"]
		if ok {
			if itemId == float64(entityId) {
				json.NewEncoder(w).Encode(item)
				return
			}
		}
	}

	RespondERR(w, http.StatusNotFound, ElementNotFound)
	return

}

func (h *Handler) Save(entity string, w http.ResponseWriter, r *http.Request) {
	body := make(map[string]interface{})
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		RespondERR(w, http.StatusBadRequest, InvalidBody)
	}
	value := h.db[entity]
	value = append(value, body)
	h.db[entity] = value

	if err := h.writeDB(); err != nil {
		RespondERR(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(h.db)
	return
}

func (h *Handler) RemoveById(entity string, w http.ResponseWriter, r *http.Request) {
	entityId, err := strconv.Atoi(chi.URLParam(r, "entityId"))
	if err != nil {
		RespondERR(w, http.StatusBadRequest, InvalidId)
		return
	}
	value := h.db[entity]
	value, err = removeElement(value, float64(entityId))
	if err != nil {
		RespondERR(w, http.StatusNotFound, err.Error())
		return
	}

	if err := h.writeDB(); err != nil {
		RespondERR(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
	json.NewEncoder(w).Encode(h.db)
}

func (h *Handler) Update(entity string, w http.ResponseWriter, r *http.Request) {
	entityId, err := strconv.Atoi(chi.URLParam(r, "entityId"))
	if err != nil {
		RespondERR(w, http.StatusBadRequest, InvalidId)
		return
	}

	body := make(map[string]interface{})
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		RespondERR(w, http.StatusBadRequest, InvalidBody)
		return
	}
	value := h.db[entity]
	found := false
	for _, data := range value {
		if data["id"] == float64(entityId) {
			found = true
			for key := range data {
				if key != "id" {
					data[key] = body[key]
				}
			}
			break
		}
	}

	if !found {
		RespondERR(w, http.StatusNotFound, ElementNotFound)
		return
	}

	if err := h.writeDB(); err != nil {
		RespondERR(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// writes a json to the response writter object
func RespondERR(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorDto{Message: message})
}

// remove an element from a slice
func removeElement(slice []map[string]interface{}, entityId float64) ([]map[string]interface{}, error) {
	nSlice := make([]map[string]interface{}, 0)
	for _, data := range slice {
		if data["id"] != entityId {
			nSlice = append(nSlice, data)
		}
	}
	if len(nSlice) == len(slice) {
		return nil, errors.New(ElementNotFound)
	}

	return nSlice, nil
}

// intercept request and add content type header to it
func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// midleware to handle undefined route/endpoint
func handleNotFound(w http.ResponseWriter, r *http.Request) {
	RespondERR(w, http.StatusNotFound, "endpoint not found")
	return
}
