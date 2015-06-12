package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func writeInvalidJSONResponse(w http.ResponseWriter, err error) {
	writeJSONResponse(w, http.StatusBadRequest, HandlerError{
		Error: err.Error(),
	})
}

func writeInternalErrorJSONResponse(w http.ResponseWriter, err error) {
	writeJSONResponse(w, http.StatusInternalServerError, HandlerError{
		Error: err.Error(),
	})
}

func writeStatusOKResponse(w http.ResponseWriter, jsonObj interface{}) {
	writeJSONResponse(w, http.StatusOK, jsonObj)
}

func writeJSONResponse(w http.ResponseWriter, statusCode int, jsonObj interface{}) {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if jsonObj != nil {
		jsonBytes, err := json.Marshal(jsonObj)
		if err != nil {
			panic("Unable to encode JSON: " + err.Error())
		}
		w.Write(jsonBytes)
		w.Header().Set("Content-Length", strconv.Itoa(len(jsonBytes)))
	}
}

type HandlerError struct {
	Error string `json:"error"`
}
