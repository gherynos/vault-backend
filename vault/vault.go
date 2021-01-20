package vault

import (
	"errors"
	"fmt"
	s "github.com/gherynos/vault-backend/store"
	"github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type Vault struct {
	roleId, secretId, prefix string
	client                   *api.Client

	tokenExpiration time.Time

	m sync.Mutex
}

func NewWithToken(vaultURL, token, prefix string) (*Vault, error) {

	var v Vault
	var err error
	if v.client, err = api.NewClient(&api.Config{Address: vaultURL}); err != nil {

		return nil, err
	}

	v.prefix = prefix
	v.client.SetToken(token)

	return &v, nil
}

func NewWithAppRole(vaultURL, roleId, secretId, prefix string) (*Vault, error) {

	var v Vault
	var err error
	if v.client, err = api.NewClient(&api.Config{Address: vaultURL}); err != nil {

		return nil, err
	}

	v.roleId = roleId
	v.secretId = secretId
	v.prefix = prefix

	if err = v.authenticate(); err != nil {

		return nil, err
	}

	return &v, nil
}

func (v *Vault) authenticate() error {

	options := map[string]interface{}{
		"role_id":   v.roleId,
		"secret_id": v.secretId,
	}

	if secret, err := v.client.Logical().Write("auth/approle/login", options); err != nil {

		return err

	} else {

		v.client.SetToken(secret.Auth.ClientToken)
		v.tokenExpiration = time.Now()
		if secret.Auth.Renewable {

			v.tokenExpiration = v.tokenExpiration.Add(time.Duration(secret.Auth.LeaseDuration-60) * time.Second)
		}
	}

	return nil
}

func (v *Vault) refreshToken() error {

	// only refresh the token when using AppRole
	if v.roleId == "" && v.secretId == "" {

		return nil
	}

	// re-authenticate if the token has expired
	v.m.Lock()
	if v.tokenExpiration.Before(time.Now()) {

		log.Debug("Refreshing Vault token...")

		if err := v.authenticate(); err != nil {

			v.m.Unlock()
			return err
		}
	}
	v.m.Unlock()

	return nil
}

func (v *Vault) Set(name, data string) error {

	if err := v.refreshToken(); err != nil {

		return err
	}

	if _, err := v.client.Logical().Write(fmt.Sprintf("secret/data/%s/%s", v.prefix, name),
		map[string]interface{}{"data": map[string]interface{}{"value": data}}); err != nil {

		return err
	}

	return nil
}

func (v *Vault) SetBin(name string, data []byte) error {

	if value, err := Encode(data); err != nil {

		return err

	} else {

		return v.Set(name, value)
	}
}

func (v *Vault) Get(name string) (string, error) {

	if err := v.refreshToken(); err != nil {

		return "", err
	}

	if secret, err := v.client.Logical().Read(fmt.Sprintf("secret/data/%s/%s", v.prefix, name)); err != nil {

		return "", err

	} else {

		if secret == nil {

			return "", &s.ItemNotFoundError{}
		}

		if data, err := secret.Data["data"].(map[string]interface{}); !err {

			return "", errors.New("unable to convert secret data")

		} else {

			return data["value"].(string), nil
		}
	}
}

func (v *Vault) GetBin(name string) (out []byte, err error) {

	if value, err := v.Get(name); err != nil {

		return nil, err

	} else {

		out, err = Decode(value)
	}
	return
}

func (v *Vault) Delete(name string) error {

	if err := v.refreshToken(); err != nil {

		return err
	}

	if _, err := v.client.Logical().Delete(fmt.Sprintf("secret/metadata/%s/%s", v.prefix, name)); err != nil {

		return err
	}

	return nil
}
