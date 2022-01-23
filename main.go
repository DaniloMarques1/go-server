package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	//entity string
	db         map[string]interface{}
	router     *chi.Mux
	fileName   string
	serverPort int
}

func NewHandler() (*Handler, error) {
	handler := Handler{}
	handler.setFlags()
	if err := handler.setDb(); err != nil {
		return nil, err
	}
	handler.router = chi.NewRouter()
	return &handler, nil
}

type ErrorDto struct {
	Message string `json:"message"`
}

func main() {
	handler, err := NewHandler()
	if err != nil {
		log.Fatal(err)
	}
	handler.router.Use(middleware)
	handler.router.NotFound(handleNotFound)

	fmt.Println("Resources available")
	fmt.Println("-----------------------------------------------------------------------")
	for entity := range handler.db {
		resources(entity, handler.serverPort)
		handler.registerRoutes(entity)
	}

	fmt.Println()
	log.Printf("Starting server on port %v\n", handler.serverPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", handler.serverPort), handler.router))
}

// set the serverPort and fileName
func (h *Handler) setFlags() {
	var portFlag int
	flag.IntVar(&portFlag, "p", 8080, "Defined server port")
	flag.IntVar(&portFlag, "port", 8080, "Defined server port")

	var fileFlag string
	flag.StringVar(&fileFlag, "watch", "db.json", "Defines which file will represent the api database")

	flag.Parse()

	h.serverPort = portFlag
	h.fileName = fileFlag
}

// read the db.json file and set the db's map
func (h *Handler) setDb() error {
	bytes, err := os.ReadFile(h.fileName)
	if err != nil {
		return err
	}

	db := make(map[string]interface{})
	err = json.Unmarshal(bytes, &db)
	if err != nil {
		return err
	}

	h.db = db
	return nil
}

// it will print the resources available
func resources(entity string, port int) {
	baseUrl := fmt.Sprintf("http://localhost:%v", port)
	fmt.Printf("%v/%v\n", baseUrl, entity)
}

// create REST endpoints for all the entities defined
// on the db.json file
func (h *Handler) registerRoutes(entity string) {
	h.router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(h.db)
	})

	h.router.Get(fmt.Sprintf("/%v", entity), func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{entity: h.db[entity]})
	})

	h.router.Get(fmt.Sprintf("/%v/{entityId}", entity), func(w http.ResponseWriter, r *http.Request) {
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
	})

	h.router.Post(fmt.Sprintf("/%v", entity), func(w http.ResponseWriter, r *http.Request) {
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
	})

	h.router.Delete(fmt.Sprintf("/%v/{entityId}", entity), func(w http.ResponseWriter, r *http.Request) {
		entityId, err := strconv.Atoi(chi.URLParam(r, "entityId"))
		if err != nil {
			RespondJSON(w, http.StatusBadRequest, "Invalid id")
			return
		}
		value := h.db[entity]

		switch value.(type) {
		case []interface{}:
			// TODO need to traverse the slice looking for entityId to remove from the slice
			fmt.Println("It is an array")
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
	})
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

func RespondJSON(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorDto{Message: message})
}

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func handleNotFound(w http.ResponseWriter, r *http.Request) {
	RespondJSON(w, http.StatusNotFound, "endpoint not found")
	return
}
