package server

import (
	"encoding/base64"
	"strings"
	"sync"

	s "github.com/gherynos/vault-backend/store"
	"github.com/gherynos/vault-backend/vault"
	log "github.com/sirupsen/logrus"
)

// VaultPool is an implementation of Pool that manages Vault stores.
type VaultPool struct {
	vaultURL, prefix, store string

	stores map[string]*vault.Vault
	mutex  sync.Mutex
}

// NewVaultPool creates a new pool of Vault stores.
// VaultURL is the URL of the Vault server to connect to.
// prefix is the string prefix used when storing the secrets in Vault.
func NewVaultPool(vaultURL, prefix, store string) s.Pool {

	vp := &VaultPool{vaultURL: vaultURL, prefix: prefix, store: store}
	vp.stores = make(map[string]*vault.Vault)

	return vp
}

// Get creates or retrieves a Vault store given an identifier.
func (vp *VaultPool) Get(identifier string) (val s.Store, err error) {

	vp.mutex.Lock()
	defer vp.mutex.Unlock()

	var ok bool
	if val, ok = vp.stores[identifier]; ok {

		return
	}

	log.Debug("Creating a new Vault client...")

	var dec []byte
	if dec, err = base64.StdEncoding.DecodeString(identifier); err != nil {

		return
	}

	userPass := strings.Split(string(dec), ":")
	var vt *vault.Vault
	if userPass[0] == "TOKEN" {
		vt, err = vault.NewWithToken(vp.vaultURL, userPass[1], vp.prefix, vp.store)
	} else {
		vt, err = vault.NewWithAppRole(vp.vaultURL, userPass[0], userPass[1], vp.prefix, vp.store)
	}
	if err != nil {

		return
	}

	val = vt
	vp.stores[identifier] = vt
	return
}

// Delete removes the Vault store associated with the identifier.
// Invoking delete using a non-existing identifier has no effect.
func (vp *VaultPool) Delete(identifier string) {

	vp.mutex.Lock()
	defer vp.mutex.Unlock()

	delete(vp.stores, identifier)
}
