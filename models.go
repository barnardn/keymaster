package keymaster

/*

Data models uses in the keymaster service application.

*/

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/nu7hatch/gouuid"
	"time"
)

type AppIdentifier struct {
	Id            int64
	AppName       string
	CredentialsId int64
	DeletedAt     time.Time
}

type Credentials struct {
	Id        int
	AppNames  []AppIdentifier
	Keys      []AppKey
	CypherKey string
	DeletedAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AppKey struct {
	Id            int64
	Name          string
	Info          string
	CredentialsId int64
	DeletedAt     time.Time
}

// Credentials methods

func (cred *Credentials) GenerateCypherKey() {
	cypherUUID, _ := uuid.NewV4()
	cred.CypherKey = cypherUUID.String()
}

func (cred *Credentials) String() (s string) {

	var plainTextInfo = make(map[string]string, len(cred.Keys))
	for _, appKey := range cred.Keys {
		plainTextInfo[appKey.Name] = appKey.Info
	}
	bytes, _ := json.Marshal(plainTextInfo)
	aesKey, _ := uuid.ParseHex(cred.CypherKey)
	var iv = aesKey[:aes.BlockSize]
	cryptBuf := make([]byte, len(bytes))
	err := encryptAESCFB(cryptBuf, bytes, []byte(aesKey[:]), iv)
	if err != nil {
		panic(fmt.Sprintf("Unable to encrypt info: %v", err))
	}
	var response = map[string]string{
		"cypherKey":  cred.CypherKey,
		"cypherText": base64.StdEncoding.EncodeToString(cryptBuf),
	}
	bytes, err = json.Marshal(response)
	return string(bytes[:])
}

func (cred *Credentials) FindByAppIdentifier(db gorm.DB, appId string) (found bool, err error) {

	var (
		aid            AppIdentifier
		appIdentifiers []AppIdentifier
		keys           []AppKey
	)
	db.Find(&aid, &AppIdentifier{AppName: appId})
	if db.NewRecord(aid) {
		return false, errors.New("Not Found")
	}
	db.Model(&aid).Related(cred).Related(&keys).Related(&appIdentifiers)
	if db.NewRecord(cred) {
		return false, errors.New("Application id exits but credentials are missing or removed")
	}
	cred.AppNames = appIdentifiers
	cred.Keys = keys
	return true, nil
}

// App Identifier Methods

func (ai AppIdentifier) String() string {
	return ai.AppName
}

// Private helper methods

func encryptAESCFB(dst, src, key, iv []byte) error {
	aesBlockEncrypter, err := aes.NewCipher([]byte(key))
	if err != nil {
		return err
	}
	aesEncrypter := cipher.NewCFBEncrypter(aesBlockEncrypter, iv)
	aesEncrypter.XORKeyStream(dst, src)
	return nil
}

func DecryptAESCFB(dst, src, key, iv []byte) error {
	aesBlockDecrypter, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil
	}
	aesDecrypter := cipher.NewCFBDecrypter(aesBlockDecrypter, iv)
	aesDecrypter.XORKeyStream(dst, src)
	return nil
}
