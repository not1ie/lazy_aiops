package cmdb

import (
	"strings"

	"github.com/lazyautoops/lazy-auto-ops/internal/security"
)

func EncryptCredentialFields(secretKey string, cred *Credential) error {
	if cred == nil {
		return nil
	}
	var err error
	if cred.Password, err = security.Encrypt(secretKey, "cmdb.credential.password", cred.Password); err != nil {
		return err
	}
	if cred.PrivateKey, err = security.Encrypt(secretKey, "cmdb.credential.private_key", cred.PrivateKey); err != nil {
		return err
	}
	if cred.Passphrase, err = security.Encrypt(secretKey, "cmdb.credential.passphrase", cred.Passphrase); err != nil {
		return err
	}
	if cred.AccessKey, err = security.Encrypt(secretKey, "cmdb.credential.access_key", cred.AccessKey); err != nil {
		return err
	}
	if cred.SecretKey, err = security.Encrypt(secretKey, "cmdb.credential.secret_key", cred.SecretKey); err != nil {
		return err
	}
	return nil
}

func DecryptCredentialFields(secretKey string, cred *Credential) error {
	if cred == nil {
		return nil
	}
	var err error
	if cred.Password, err = security.Decrypt(secretKey, "cmdb.credential.password", cred.Password); err != nil {
		return err
	}
	if cred.PrivateKey, err = security.Decrypt(secretKey, "cmdb.credential.private_key", cred.PrivateKey); err != nil {
		return err
	}
	if cred.Passphrase, err = security.Decrypt(secretKey, "cmdb.credential.passphrase", cred.Passphrase); err != nil {
		return err
	}
	if cred.AccessKey, err = security.Decrypt(secretKey, "cmdb.credential.access_key", cred.AccessKey); err != nil {
		return err
	}
	if cred.SecretKey, err = security.Decrypt(secretKey, "cmdb.credential.secret_key", cred.SecretKey); err != nil {
		return err
	}
	return nil
}

func DecryptCredentialField(secretKey, field, value string) (string, error) {
	scope := "cmdb.credential." + strings.TrimSpace(field)
	return security.Decrypt(secretKey, scope, value)
}

func SanitizeCredentialFields(cred *Credential) {
	if cred == nil {
		return
	}
	cred.Password = ""
	cred.PrivateKey = ""
	cred.Passphrase = ""
	cred.AccessKey = ""
	cred.SecretKey = ""
}
