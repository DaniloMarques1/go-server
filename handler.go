package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	db         map[string]interface{}
	router     *chi.Mux
	fileName   string
	serverPort int
}

func NewHandler(fileName string, serverPort int) (*Handler, error) {
	router := chi.NewRouter()
	handler := &Handler{
		router:     router,
		fileName:   fileName,
		serverPort: serverPort,
	}

	db, err := handler.readDb()
	if err != nil {
		return nil, err
	}
	handler.db = db

	return handler, nil
}

// read the file given as argument
func (h *Handler) readDb() (map[string]interface{}, error) {
	bytes, err := os.ReadFile(h.fileName)
	if err != nil {
		return nil, err
	}

	db := make(map[string]interface{})
	err = json.Unmarshal(bytes, &db)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// write te current db state on file
func (h *Handler) writeDB() error {
	bytes, err := json.MarshalIndent(h.db, "", "  ")
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
func (h *Handler) registerRoutes(entity string) {
	h.router.Get(fmt.Sprintf("/%v", entity), func(w http.ResponseWriter, r *http.Request) {
		h.FindAll(entity, w, r)
		return
	})

	h.router.Get(fmt.Sprintf("/%v/{entityId}", entity), func(w http.ResponseWriter, r *http.Request) {
		h.FindById(entity, w, r)
		return
	})

	h.router.Post(fmt.Sprintf("/%v", entity), func(w http.ResponseWriter, r *http.Request) {
		h.Save(entity, w, r)
		return
	})

	h.router.Delete(fmt.Sprintf("/%v/{entityId}", entity), func(w http.ResponseWriter, r *http.Request) {
		h.RemoveById(entity, w, r)
		return
	})

	h.router.Put(fmt.Sprintf("/%v/{entityId}", entity), func(w http.ResponseWriter, r *http.Request) {
		h.Update(entity, w, r)
		return
	})
}

func (h *Handler) FindAll(entity string, w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{entity: h.db[entity]})
}

func (h *Handler) FindById(entity string, w http.ResponseWriter, r *http.Request) {
	value := h.db[entity]
	entityId, err := strconv.Atoi(chi.URLParam(r, "entityId"))
	if err != nil {
		RespondJSON(w, http.StatusBadRequest, "Invalid id")
		return
	}
	switch value.(type) {
	case string:
		json.NewEncoder(w).Encode(value)
		return
	case []interface{}:
		arr := value.([]interface{})
		for _, item := range arr {
			obj := item.(map[string]interface{})
			objId := obj["id"].(float64)
			if int(objId) == entityId {
				json.NewEncoder(w).Encode(obj)
				return
			}
		}
		RespondJSON(w, http.StatusBadRequest, "Not found")
		return
	case map[string]interface{}:
		obj := value.(map[string]interface{})
		objId := obj["id"].(float64)
		if int(objId) == entityId {
			json.NewEncoder(w).Encode(obj)
			return
		}
		RespondJSON(w, http.StatusBadRequest, "Not found")
		return
	default:
		log.Println("No type matched")
	}

	RespondJSON(w, http.StatusBadRequest, "Something went wrong")
	return

}

func (h *Handler) Save(entity string, w http.ResponseWriter, r *http.Request) {
	body := make(map[string]interface{})
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		RespondJSON(w, http.StatusBadRequest, "Invalid body")
	}
	value := h.db[entity]

	switch value.(type) {
	case []interface{}:
		arr := value.([]interface{})
		arr = append(arr, body)
		h.db[entity] = arr
	case map[string]interface{}:
		h.db[entity] = body
	default:
		// TODO add some return here
		return
	}

	if err := h.writeDB(); err != nil {
		RespondJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(h.db)
}

func (h *Handler) RemoveById(entity string, w http.ResponseWriter, r *http.Request) {
	entityId, err := strconv.Atoi(chi.URLParam(r, "entityId"))
	if err != nil {
		RespondJSON(w, http.StatusBadRequest, "Invalid id")
		return
	}
	value := h.db[entity]

	switch value.(type) {
	case []interface{}:
		slice := value.([]interface{})
		slice = removeElement(slice, float64(entityId))
		fmt.Println(slice)
		h.db[entity] = slice
	case map[string]interface{}:
		obj, ok := value.(map[string]interface{})
		if !ok {
			RespondJSON(w, http.StatusBadRequest, "Something went wrong")
			return
		}
		if len(obj) == 0 {
			RespondJSON(w, http.StatusNotFound, "Not found")
			return
		}

		objId := obj["id"].(float64)
		if int(objId) == entityId {
			h.db[entity] = map[string]interface{}{}
		}
	default:
		// TODO add return
		return
	}

	if err := h.writeDB(); err != nil {
		RespondJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(h.db)
}

func (h *Handler) Update(entity string, w http.ResponseWriter, r *http.Request) {
	entityId, err := strconv.Atoi(chi.URLParam(r, "entityId"))
	if err != nil {
		RespondJSON(w, http.StatusBadRequest, "Invalid id")
		return
	}

	body := make(map[string]interface{})
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		RespondJSON(w, http.StatusBadRequest, "Invalid body")
		return
	}
	entityData := h.db[entity] // here we have the person array
	entitySlice, ok := entityData.([]interface{})
	if !ok {
		RespondJSON(w, http.StatusBadRequest, "Invalid entity") // TODO better message for the problem
		return
	}
	for _, data := range entitySlice {
		entityObj, ok := data.(map[string]interface{})
		if ok {
			if entityObj["id"] == float64(entityId) {
				entityObj["name"] = body["name"]
				entityObj["age"] = body["age"]
				break
			}
		}
	}
	if err := h.writeDB(); err != nil {
		RespondJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
}

// remove an element from a slice
func removeElement(slice []interface{}, entityId float64) []interface{} {
	nSlice := make([]interface{}, 0)
	for _, data := range slice {
		entityObj, ok := data.(map[string]interface{})
		if ok {
			if entityObj["id"] != entityId {
				nSlice = append(nSlice, entityObj)
			}
		}
	}
	return nSlice
}