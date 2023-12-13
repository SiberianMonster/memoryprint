package handlersfunc

import (
	//"context"
	"encoding/json"
	"github.com/SiberianMonster/memoryprint/internal/config"
	"log"
	"net/http"
    "errors"
    "fmt"
    "io"
    "strings"
)


func UserIDContextReader(r *http.Request) (uint) {

	userID := r.Context().Value(config.UserIDKey).(uint)
	return userID
}

func HandleWrongCredentialsError(rw http.ResponseWriter, resp map[string]string) {
    rw.WriteHeader(http.StatusNoContent)
    resp["status"] = "incorrect credentials error"
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandlePermissionError(rw http.ResponseWriter, resp map[string]string) {
    rw.WriteHeader(http.StatusUnauthorized)
    resp["status"] = "action not permitted"
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleUsernameAlreadyTaken(rw http.ResponseWriter, resp map[string]string) {
    rw.WriteHeader(http.StatusConflict)
    resp["status"] = "username already taken"
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleNoContent(rw http.ResponseWriter, resp map[string]string) {
    rw.WriteHeader(http.StatusNoContent)
    resp["status"] = "no content found"
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleWrongBytesInput(rw http.ResponseWriter, resp map[string]string) {
    rw.WriteHeader(http.StatusNoContent)
    resp["status"] = "wrong bytes input"
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleDatabaseServerError(rw http.ResponseWriter, resp map[string]string) {
		rw.WriteHeader(http.StatusInternalServerError)
		resp["status"] = "sql database error"
		jsonResp, err := json.Marshal(resp)
		if err != nil {
			log.Printf("Error happened in JSON marshal. Err: %s", err)
			return
		}
		rw.Write(jsonResp)
}

func HandleJWTError(rw http.ResponseWriter, resp map[string]string) {
    rw.WriteHeader(http.StatusInternalServerError)
    resp["status"] = "jwt tokenizer error"
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleDecodeError(rw http.ResponseWriter, resp map[string]string, err error) {

    if err != nil {
        var syntaxError *json.SyntaxError
        var unmarshalTypeError *json.UnmarshalTypeError
        var msg string

        switch {
        case errors.As(err, &syntaxError):
            msg = fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)

        case errors.Is(err, io.ErrUnexpectedEOF):
            msg = fmt.Sprintf("Request body contains badly-formed JSON")

        case errors.As(err, &unmarshalTypeError):
            msg = fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)

        case strings.HasPrefix(err.Error(), "json: unknown field "):
            fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
            msg = fmt.Sprintf("Request body contains unknown field %s", fieldName)

        case errors.Is(err, io.EOF):
            msg = "Request body must not be empty"

        case err.Error() == "http: request body too large":
            msg = "Request body must not be larger than 1MB"

        default:
            msg = err.Error()
        }
        rw.WriteHeader(http.StatusBadRequest)
        resp["status"] = msg
        jsonResp, err := json.Marshal(resp)
        if err != nil {
                log.Printf("Error happened in JSON marshal. Err: %s", err)
                return
        }
        rw.Write(jsonResp)
    }

}

func HandleMailSendError(rw http.ResponseWriter, resp map[string]string) {
    rw.WriteHeader(http.StatusInternalServerError)
    resp["status"] = "unable to send email"
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleVerificationError(rw http.ResponseWriter, resp map[string]string) {
    rw.WriteHeader(http.StatusInternalServerError)
    resp["status"] = "code verification error"
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandlePromocodeError(rw http.ResponseWriter, resp map[string]string, err error) {

    if err != nil {

        rw.WriteHeader(http.StatusBadRequest)
        resp["status"] = err.Error() 
        jsonResp, err := json.Marshal(resp)
        if err != nil {
                log.Printf("Error happened in JSON marshal. Err: %s", err)
                return
        }
        rw.Write(jsonResp)
    }

}