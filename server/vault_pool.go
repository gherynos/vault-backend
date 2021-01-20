package server

import (
	"encoding/base64"
	s "github.com/gherynos/vault-backend/store"
	"github.com/gherynos/vault-backend/vault"
	log "github.com/sirupsen/logrus"
	"strings"
	"sync"
)

type VaultPool struct {
	vaultURL, prefix string

	stores map[string]*vault.Vault
	mutex  sync.Mutex
}

func NewVaultPool(vaultURL, prefix string) s.Pool {

	vp := &VaultPool{vaultURL: vaultURL, prefix: prefix}
	vp.stores = make(map[string]*vault.Vault)

	return vp
}

func (vp *VaultPool) Get(identifier string) (val s.Store, err error) {

	vp.mutex.Lock()
	defer vp.mutex.Unlock()

	var ok bool
	if val, ok = vp.stores[identifier]; ok {

		return

	} else {

		log.Debug("Creating a new Vault client...")

		var dec []byte
		if dec, err = base64.StdEncoding.DecodeString(identifier); err != nil {

			return
		}

		userPass := strings.Split(string(dec), ":")
		var vt *vault.Vault
		if userPass[0] == "TOKEN" {

			vt, err = vault.NewWithToken(vp.vaultURL, userPass[1], vp.prefix)

		} else {

			vt, err = vault.NewWithAppRole(vp.vaultURL, userPass[0], userPass[1], vp.prefix)
		}
		if err != nil {

			return
		}

		val = vt
		vp.stores[identifier] = vt
		return
	}
}

func (vp *VaultPool) Delete(identifier string) {

	vp.mutex.Lock()
	defer vp.mutex.Unlock()

	delete(vp.stores, identifier)
}
