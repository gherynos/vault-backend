package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	s "github.com/gherynos/vault-backend/store"
	"github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
)

// Version defines the version of the server
const Version = "0.4.3"

func checkLockID(store s.Store, state, id string) (proceed bool, data string, err error) {

	var value []byte
	if value, err = store.GetBin(fmt.Sprintf("%s-lock", state)); err != nil {

		proceed = false
		return
	}
	data = string(value)

	var jData map[string]interface{}
	if err = json.Unmarshal(value, &jData); err != nil {

		return
	}

	proceed = jData["ID"] == id
	return
}

func stateHandlerGet(logger *log.Entry, store s.Store, state string, w http.ResponseWriter) (int, string) {

	logger.Debug("Load state")

	data, err := store.GetBin(state)
	if err != nil {

		var itemNotFoundError *s.ItemNotFoundError
		var responseError *api.ResponseError
		switch {

		case errors.As(err, &itemNotFoundError):
			return http.StatusNotFound, http.StatusText(http.StatusNotFound)
		case errors.As(err, &responseError):
			{
				return responseError.StatusCode, responseError.Error()
			}
		default:
			{
				logger.WithError(err).Error("unable to get state")
				return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := io.Copy(w, bytes.NewReader(data)); err != nil {

		logger.WithError(err).Error("unable to return state")
		return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)
	}

	return 200, ""
}

func stateHandlerPost(logger *log.Entry, store s.Store, state string, r *http.Request, w http.ResponseWriter) (int, string) {

	logger.Debug("Store state")

	if proceed, data, err := checkLockID(store, state, r.URL.Query().Get("ID")); err != nil {

		var itemNotFoundError *s.ItemNotFoundError
		var responseError *api.ResponseError
		switch {

		case errors.As(err, &itemNotFoundError):
			return http.StatusUnprocessableEntity, http.StatusText(http.StatusUnprocessableEntity)
		case errors.As(err, &responseError):
			{
				return responseError.StatusCode, responseError.Error()
			}
		default:
			{
				logger.WithError(err).Error("unable to check lock")
				return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)
			}
		}

	} else if !proceed {

		w.Header().Set("Content-Type", "application/json")
		return http.StatusLocked, data
	}

	var reqBody []byte
	var err error
	if reqBody, err = io.ReadAll(r.Body); err != nil {

		return http.StatusBadRequest, http.StatusText(http.StatusBadRequest)
	}

	if err := store.SetBin(state, reqBody); err != nil {

		var responseError *api.ResponseError
		switch {

		case errors.As(err, &responseError):
			{
				return responseError.StatusCode, responseError.Error()
			}
		default:
			{
				logger.WithError(err).Error("unable to store state")
				return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)
			}
		}
	}

	return 200, ""
}

func stateHandlerLock(logger *log.Entry, store s.Store, state string, r *http.Request, w http.ResponseWriter) (int, string) {

	logger.Debug("Lock state")

	name := fmt.Sprintf("%s-lock", state)
	data, err := store.GetBin(name)
	if err != nil {

		var itemNotFoundError *s.ItemNotFoundError
		var responseError *api.ResponseError
		switch {

		case errors.As(err, &itemNotFoundError):
			{
				var reqBody []byte
				var err error
				if reqBody, err = io.ReadAll(r.Body); err != nil {

					return http.StatusBadRequest, http.StatusText(http.StatusBadRequest)
				}

				if err := store.SetBin(name, reqBody); err != nil {

					logger.WithError(err).Error("unable to store lock")
					return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)
				}

				return 200, ""
			}
		case errors.As(err, &responseError):
			{
				return responseError.StatusCode, responseError.Error()
			}
		default:
			{
				logger.WithError(err).Error("unable to retrieve lock")
				return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	return http.StatusConflict, string(data)
}

func stateHandlerUnlock(logger *log.Entry, store s.Store, state string, pool s.Pool, userPassEnc string, r *http.Request, w http.ResponseWriter) (int, string) {

	logger.Debug("Unlock state")

	var reqBody []byte
	var err error
	if reqBody, err = io.ReadAll(r.Body); err != nil {

		return http.StatusBadRequest, http.StatusText(http.StatusBadRequest)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(reqBody, &body); err != nil {

		return http.StatusBadRequest, http.StatusText(http.StatusBadRequest)
	}

	if proceed, data, err := checkLockID(store, state, body["ID"].(string)); err != nil {

		var itemNotFoundError *s.ItemNotFoundError
		var responseError *api.ResponseError
		switch {

		case errors.As(err, &itemNotFoundError):
			return http.StatusUnprocessableEntity, http.StatusText(http.StatusUnprocessableEntity)
		case errors.As(err, &responseError):
			{
				return responseError.StatusCode, responseError.Error()
			}
		default:
			{
				logger.WithError(err).Error("unable to check lock")
				return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)
			}
		}

	} else if !proceed {

		w.Header().Set("Content-Type", "application/json")
		return http.StatusConflict, data
	}

	if err := store.Delete(fmt.Sprintf("%s-lock", state)); err != nil {

		var responseError *api.ResponseError
		switch {

		case errors.As(err, &responseError):
			{
				return responseError.StatusCode, responseError.Error()
			}
		default:
			{
				logger.WithError(err).Error("unable to remove lock")
				return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)
			}
		}
	}

	pool.Delete(userPassEnc)

	return 200, ""
}

func stateHandler(pool s.Pool, w http.ResponseWriter, r *http.Request) (int, string) {

	state := r.URL.Path[7:] // /state/...

	logger := log.WithFields(log.Fields{"state": state})

	userPassEnc := r.Header.Get("Authorization")
	if userPassEnc == "" {

		return http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized)
	}
	userPassEnc = userPassEnc[6:] // Basic ...

	var store s.Store
	var err error
	store, err = pool.Get(userPassEnc)
	if err != nil {

		var responseError *api.ResponseError
		switch {

		case errors.As(err, &responseError):
			{
				logger.Debugf("error connecting to Vault: %d - %s", responseError.StatusCode, responseError.Error())
				return responseError.StatusCode, responseError.Error()
			}
		default:
			{
				logger.WithError(err).Error("error connecting to Vault")
				return http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable)
			}
		}
	}

	switch r.Method {

	case "GET":
		{
			return stateHandlerGet(logger, store, state, w)
		}

	case "POST":
		{
			return stateHandlerPost(logger, store, state, r, w)
		}

	case "LOCK":
		{
			return stateHandlerLock(logger, store, state, r, w)
		}

	case "UNLOCK":
		{
			return stateHandlerUnlock(logger, store, state, pool, userPassEnc, r, w)
		}

	default:
		{
			logger.Warnf("Method %s not allowed", r.Method)
			return http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed)
		}
	}
}

type handler struct {
	pool s.Pool
	f    func(s.Pool, http.ResponseWriter, *http.Request) (int, string)
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if code, msg := h.f(h.pool, w, r); code != http.StatusOK {

		http.Error(w, msg, code)
	}
}

func getEnv(key, fallback string) string {

	if value, ok := os.LookupEnv(key); ok {

		return value
	}

	return fallback
}

// RunServer starts the Vault Backend TCP server
func RunServer() {

	if _, debug := os.LookupEnv("DEBUG"); debug {

		log.SetLevel(log.DebugLevel)
	}
	log.SetOutput(os.Stdout)

	vaultURL := getEnv("VAULT_URL", "http://localhost:8200")
	vaultPrefix := getEnv("VAULT_PREFIX", "vbk")
	vaultStore := getEnv("VAULT_STORE", "secret")
	address := getEnv("LISTEN_ADDRESS", ":8080")
	tlsCrt := getEnv("TLS_CRT", "")
	tlsKey := getEnv("TLS_KEY", "")

	log.Infof("Vault Backend version %s listening on %s", Version, address)
	log.Debugf("Vault URL: %s, secret prefix: %s", vaultURL, vaultPrefix)

	http.Handle("/state/", handler{NewVaultPool(vaultURL, vaultPrefix, vaultStore), stateHandler})

	if tlsCrt != "" && tlsKey != "" {

		log.Fatal(http.ListenAndServeTLS(address, tlsCrt, tlsKey, nil))

	} else {

		log.Fatal(http.ListenAndServe(address, nil))
	}
}
