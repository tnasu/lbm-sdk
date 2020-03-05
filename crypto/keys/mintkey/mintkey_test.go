package mintkey_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/crypto/bcrypt"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/armor"
	cryptoAmino "github.com/tendermint/tendermint/crypto/encoding/amino"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"github.com/tendermint/tendermint/crypto/xsalsa20symmetric"

	"github.com/cosmos/cosmos-sdk/crypto/keys"
	"github.com/cosmos/cosmos-sdk/crypto/keys/mintkey"
)

func TestArmorUnarmorPrivKey(t *testing.T) {
	priv := secp256k1.GenPrivKey()
	armored := mintkey.EncryptArmorPrivKey(priv, "passphrase", "")
	_, _, err := mintkey.UnarmorDecryptPrivKey(armored, "wrongpassphrase")
	require.Error(t, err)
	decrypted, algo, err := mintkey.UnarmorDecryptPrivKey(armored, "passphrase")
	require.NoError(t, err)
	require.Equal(t, string(keys.Secp256k1), algo)
	require.True(t, priv.Equals(decrypted))

	// empty string
	decrypted, algo, err = mintkey.UnarmorDecryptPrivKey("", "passphrase")
	require.Error(t, err)
	require.True(t, errors.Is(io.EOF, err))
	require.Nil(t, decrypted)
	require.Empty(t, algo)

	// wrong key type
	armored = mintkey.ArmorPubKeyBytes(priv.PubKey().Bytes(), "")
	decrypted, algo, err = mintkey.UnarmorDecryptPrivKey(armored, "passphrase")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unrecognized armor type")

	// armor key manually
	encryptPrivKeyFn := func(privKey crypto.PrivKey, passphrase string) (saltBytes []byte, encBytes []byte) {
		saltBytes = crypto.CRandBytes(16)
		key, err := bcrypt.GenerateFromPassword(saltBytes, []byte(passphrase), mintkey.BcryptSecurityParameter)
		require.NoError(t, err)
		key = crypto.Sha256(key) // get 32 bytes
		privKeyBytes := privKey.Bytes()
		return saltBytes, xsalsa20symmetric.EncryptSymmetric(privKeyBytes, key)
	}
	saltBytes, encBytes := encryptPrivKeyFn(priv, "passphrase")

	// wrong kdf header
	headerWrongKdf := map[string]string{
		"kdf":  "wrong",
		"salt": fmt.Sprintf("%X", saltBytes),
		"type": "secp256k",
	}
	armored = armor.EncodeArmor("TENDERMINT PRIVATE KEY", headerWrongKdf, encBytes)
	_, _, err = mintkey.UnarmorDecryptPrivKey(armored, "passphrase")
	require.Error(t, err)
	require.Equal(t, "unrecognized KDF type: wrong", err.Error())
}

func TestArmorUnarmorPubKey(t *testing.T) {
	// Select the encryption and storage for your cryptostore
	cstore := keys.NewInMemory()

	// Add keys and see they return in alphabetical order
	info, _, err := cstore.CreateMnemonic("Bob", keys.English, "passphrase", keys.Secp256k1)
	require.NoError(t, err)
	armored := mintkey.ArmorPubKeyBytes(info.GetPubKey().Bytes(), "")
	pubBytes, algo, err := mintkey.UnarmorPubKeyBytes(armored)
	require.NoError(t, err)
	pub, err := cryptoAmino.PubKeyFromBytes(pubBytes)
	require.NoError(t, err)
	require.Equal(t, string(keys.Secp256k1), algo)
	require.True(t, pub.Equals(info.GetPubKey()))

	armored = mintkey.ArmorPubKeyBytes(info.GetPubKey().Bytes(), "unknown")
	pubBytes, algo, err = mintkey.UnarmorPubKeyBytes(armored)
	require.NoError(t, err)
	pub, err = cryptoAmino.PubKeyFromBytes(pubBytes)
	require.NoError(t, err)
	require.Equal(t, "unknown", algo)
	require.True(t, pub.Equals(info.GetPubKey()))

	// armor pubkey manually
	header := map[string]string{
		"version": "0.0.0",
		"type":    "unknown",
	}
	armored = armor.EncodeArmor("TENDERMINT PUBLIC KEY", header, pubBytes)
	_, algo, err = mintkey.UnarmorPubKeyBytes(armored)
	require.NoError(t, err)
	// return secp256k1 if version is 0.0.0
	require.Equal(t, "secp256k1", algo)

	// missing version header
	header = map[string]string{
		"type": "unknown",
	}
	armored = armor.EncodeArmor("TENDERMINT PUBLIC KEY", header, pubBytes)
	bz, algo, err := mintkey.UnarmorPubKeyBytes(armored)
	require.Nil(t, bz)
	require.Empty(t, algo)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unrecognized version")
}

func TestArmorInfoBytes(t *testing.T) {
	bs := []byte("test")
	armoredString := mintkey.ArmorInfoBytes(bs)
	unarmoredBytes, err := mintkey.UnarmorInfoBytes(armoredString)
	require.NoError(t, err)
	require.True(t, bytes.Equal(bs, unarmoredBytes))
}

func TestUnarmorInfoBytesErrors(t *testing.T) {
	unarmoredBytes, err := mintkey.UnarmorInfoBytes("")
	require.Error(t, err)
	require.True(t, errors.Is(io.EOF, err))
	require.Nil(t, unarmoredBytes)

	header := map[string]string{
		"type":    "Info",
		"version": "0.0.1",
	}
	unarmoredBytes, err = mintkey.UnarmorInfoBytes(armor.EncodeArmor(
		"TENDERMINT KEY INFO", header, []byte("plain-text")))
	require.Error(t, err)
	require.Equal(t, "unrecognized version: 0.0.1", err.Error())
	require.Nil(t, unarmoredBytes)
}