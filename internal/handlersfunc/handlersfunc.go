package handlersfunc

import (
	//"context"
	"encoding/json"
	"github.com/SiberianMonster/memoryprint/internal/config"
    "github.com/go-playground/validator/v10"
	"log"
	"net/http"
    "errors"
    "fmt"
    "io"
    "strings"
)

type ErrorBody struct {
    ErrorCode    uint     `json:"error_code"`
    ErrorMessage    string     `json:"error_message"`
}

type ValidationErrorBody struct {
    ErrorCode    uint     `json:"error_code"`
    ErrorMessage    string     `json:"error_message"`
    Errors   map[string][]string     `json:"data"`
}

func msgForTag(tag string) string {
    switch tag {
    case "required":
        return "required"
    case "email":
        return "email"
    
    case "oneof":
        return "enum"
    case "min":
        return "length"
    case "max":
        return "length"
    case "lte":
        return "enum"
    case "gte":
        return "enum"
    case "necsfield":
        return "equalsoldpass"
    }
    return ""
}

func UserIDContextReader(r *http.Request) (uint) {

	userID := r.Context().Value(config.UserIDKey).(uint)
	return userID
}

func HandleWrongCredentialsError(rw http.ResponseWriter) {
    
    rw.WriteHeader(http.StatusOK)
    
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 401
    errorB.ErrorMessage = "Wrong password"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
   
}

func HandlePermissionError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusForbidden)
}

func HandleExpiredError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 410
    errorB.ErrorMessage = "Expired promooffer"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleAlreadyUsedError(rw http.ResponseWriter) {
    
    rw.WriteHeader(http.StatusOK)
    
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 302
    errorB.ErrorMessage = "Already used promocode"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
   
}

func HandleAlreadyUsedGiftcertificateError(rw http.ResponseWriter) {
    
    rw.WriteHeader(http.StatusOK)
    
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 301
    errorB.ErrorMessage = "Already used gift certificate"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
   
}


func HandleUnregisteredUserError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 404
    errorB.ErrorMessage = "User with this email does not exist"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
   
}

func HandleUsernameAlreadyTaken(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    var errorB ErrorBody
    resp := make(map[string]ErrorBody)
    errorB.ErrorCode = 409
    errorB.ErrorMessage = "User with this email already exists"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleNoContent(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusNoContent)
}

func HandleWrongBytesInput(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusNoContent)
}

func HandleDatabaseServerError(rw http.ResponseWriter) {
	rw.WriteHeader(http.StatusInternalServerError)
}

func HandleJWTError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusUnauthorized)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 401
    errorB.ErrorMessage = "jwt tokenizer error"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleDecodeError(rw http.ResponseWriter, err error) {

    resp := make(map[string]ErrorBody)
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
        rw.WriteHeader(http.StatusOK)
        var errorB ErrorBody
        errorB.ErrorCode = 404
        errorB.ErrorMessage = msg

        resp["error"] = errorB
        jsonResp, err := json.Marshal(resp)
        if err != nil {
                log.Printf("Error happened in JSON marshal. Err: %s", err)
                return
        }
        rw.Write(jsonResp)
    }

}

func HandleMailSendError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 424
    errorB.ErrorMessage = "unable to send email"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleMissingPageError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 404
    errorB.ErrorMessage = "One or more pages are missing"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleMissingTemplateError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusNotFound)
}

func HandleMissingProjectError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusNotFound)
    
}

func HandleCoverPageError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 405
    errorB.ErrorMessage = "Attempt to reorder a cover page"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleWrongOrderError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 503
    errorB.ErrorMessage = "Order of pages is corrupted"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleRemoveBackgroundError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 409
    errorB.ErrorMessage = "Failed to remove background for image"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleUploadImageError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 500
    errorB.ErrorMessage = "Failed to upload image to s3 bucket"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleNotAllPagesPassedError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 406
    errorB.ErrorMessage = "Not all pages passed"

    resp["error"] = errorB
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

func HandleValidationError(rw http.ResponseWriter, err error) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ValidationErrorBody)
    var errorB ValidationErrorBody
    errorB.ErrorCode = 422
    errorB.ErrorMessage = "Validation failed"
    var ve validator.ValidationErrors
    if errors.As(err, &ve) {
            out := make(map[string][]string, len(ve))
            
            for _, fe := range ve {
                log.Println(fe.Tag())
                out[strings.ToLower(fe.Field())] = []string{msgForTag(fe.Tag())}
                if strings.ToLower(fe.Field()) == "buyerphone" {
                    out[strings.ToLower(fe.Field())] = []string{"required,e164"}
                }
                if strings.ToLower(fe.Field()) == "phone" {
                    out[strings.ToLower(fe.Field())] = []string{"required,e164"}
                }
            }
            errorB.Errors = out
    }
    
    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}


func HandleSamePassError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 421
    errorB.ErrorMessage = "Same password"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleWrongImageFormatError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 415
    errorB.ErrorMessage = "Image format is not allowed"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleMissingImageDataError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 406
    errorB.ErrorMessage = "Image bytes were not passed"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleMissingPromocode(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 412
    errorB.ErrorMessage = "Promocode does not exist"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleWrongPromocodeCategoryError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 417
    errorB.ErrorMessage = "Promocode category not applicable"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleDeliveryCalculationError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 416
    errorB.ErrorMessage = "Failed to calculate delivery cost"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleWrongGiftCodeError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 414
    errorB.ErrorMessage = "Gift certificate code invalid"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleFailedPaymentURL(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 406
    errorB.ErrorMessage = "Failed to produce payment url"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleFailedCancellationError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 410
    errorB.ErrorMessage = "Failed to cancel order"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
   
}

func HandleFailedRenewSubscription(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 423
    errorB.ErrorMessage = "Failed to renew subscription"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleMissingLeatherID(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 404
    errorB.ErrorMessage = "Missing leather id"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleProjectNotPublished(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 425
    errorB.ErrorMessage = "Project not published"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleProjectPublished(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 426
    errorB.ErrorMessage = "Project published"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleOrderCompleted(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 451
    errorB.ErrorMessage = "Order already completed"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleCoverBoolError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 428
    errorB.ErrorMessage = "Hard cover unavailable"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}

func HandleCountProjectError(rw http.ResponseWriter) {
    rw.WriteHeader(http.StatusOK)
    resp := make(map[string]ErrorBody)
    var errorB ErrorBody
    errorB.ErrorCode = 413
    errorB.ErrorMessage = "Wrong count of projects"

    resp["error"] = errorB
    jsonResp, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error happened in JSON marshal. Err: %s", err)
        return
    }
    rw.Write(jsonResp)
}