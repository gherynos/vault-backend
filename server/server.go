package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	s "github.com/gherynos/vault-backend/store"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

const Version = "0.2.0"

func checkLockId(store s.Store, state, id string) (proceed bool, data string, err error) {

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

		return http.StatusUnauthorized, "Error while connecting to Vault"
	}

	switch r.Method {

	case "GET":
		{
			logger.Debug("Load state")

			data, err := store.GetBin(state)
			if err != nil {

				switch err.(type) {

				case *s.ItemNotFoundError:
					return http.StatusNotFound, http.StatusText(http.StatusNotFound)

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

	case "POST":
		{
			logger.Debug("Store state")

			if proceed, data, err := checkLockId(store, state, r.URL.Query().Get("ID")); err != nil {

				switch err.(type) {

				case *s.ItemNotFoundError:
					return http.StatusUnprocessableEntity, http.StatusText(http.StatusUnprocessableEntity)

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

			if reqBody, err := ioutil.ReadAll(r.Body); err != nil {

				return http.StatusBadRequest, http.StatusText(http.StatusBadRequest)

			} else {

				if err := store.SetBin(state, reqBody); err != nil {

					logger.WithError(err).Error("unable to store state")
					return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)
				}

				return 200, ""
			}
		}

	case "LOCK":
		{
			logger.Debug("Lock state")

			name := fmt.Sprintf("%s-lock", state)
			data, err := store.GetBin(name)
			if err != nil {

				switch err.(type) {

				case *s.ItemNotFoundError:
					{

						if reqBody, err := ioutil.ReadAll(r.Body); err != nil {

							return http.StatusBadRequest, http.StatusText(http.StatusBadRequest)

						} else {

							if err := store.SetBin(name, reqBody); err != nil {

								logger.WithError(err).Error("unable to store lock")
								return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)
							}
						}

						return 200, ""
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

	case "UNLOCK":
		{
			logger.Debug("Unlock state")

			if reqBody, err := ioutil.ReadAll(r.Body); err != nil {

				return http.StatusBadRequest, http.StatusText(http.StatusBadRequest)

			} else {

				var body map[string]interface{}
				if err := json.Unmarshal(reqBody, &body); err != nil {

					return http.StatusBadRequest, http.StatusText(http.StatusBadRequest)
				}

				if proceed, data, err := checkLockId(store, state, body["ID"].(string)); err != nil {

					switch err.(type) {

					case *s.ItemNotFoundError:
						return http.StatusUnprocessableEntity, http.StatusText(http.StatusUnprocessableEntity)

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

					logger.WithError(err).Error("unable to remove lock")
					return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)
				}

				pool.Delete(userPassEnc)

				return 200, ""
			}
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

func RunServer() {

	if _, debug := os.LookupEnv("DEBUG"); debug {

		log.SetLevel(log.DebugLevel)
	}
	log.SetOutput(os.Stdout)

	vaultURL := getEnv("VAULT_URL", "http://localhost:8200")
	vaultPrefix := getEnv("VAULT_PREFIX", "vbk")
	address := getEnv("LISTEN_ADDRESS", ":8080")
	tlsCrt := getEnv("TLS_CRT", "")
	tlsKey := getEnv("TLS_KEY", "")

	log.Infof("Vault Backend version %s listening on %s", Version, address)
	log.Debugf("Vault URL: %s, secret prefix: %s", vaultURL, vaultPrefix)

	http.Handle("/state/", handler{NewVaultPool(vaultURL, vaultPrefix), stateHandler})

	if tlsCrt != "" && tlsKey != "" {

		log.Fatal(http.ListenAndServeTLS(address, tlsCrt, tlsKey, nil))

	} else {

		log.Fatal(http.ListenAndServe(address, nil))
	}
}
