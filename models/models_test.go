package models

import (
	"os"
	"testing"

	"path"

	"reflect"

	"fmt"

	"bytes"
	"encoding/gob"

	"encoding/hex"

	"github.com/SmartMeshFoundation/Atmosphere/log"
	"github.com/SmartMeshFoundation/Atmosphere/params"
	"github.com/SmartMeshFoundation/Atmosphere/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func init() {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, utils.MyStreamHandler(os.Stderr)))
}

var dbPath string

func setupDb(t *testing.T) (model *ModelDB) {
	dbPath = path.Join(os.TempDir(), "testxxxx.db")
	os.Remove(dbPath)
	os.Remove(dbPath + ".lock")
	model, err := OpenDb(dbPath)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(model.db)
	return
}

func TestToken(t *testing.T) {
	model := setupDb(t)
	defer func() {
		model.CloseDB()
	}()
	var cbtokens []common.Address
	funcb := func(token common.Address) bool {
		cbtokens = append(cbtokens, token)
		return false
	}
	ts, err := model.GetAllTokens()
	if len(ts) > 0 {
		t.Error("should not found")
	}
	if len(ts) != 0 {
		t.Error("should be empty")
	}
	var am = make(AddressMap)
	t1 := utils.NewRandomAddress()
	am[t1] = true
	model.RegisterNewTokenCallback(funcb)
	err = model.AddToken(t1)
	if err != nil {
		t.Error(err)
	}
	am2, _ := model.GetAllTokens()
	assert.EqualValues(t, am, am2)
	t2 := utils.NewRandomAddress()
	am[t2] = true
	err = model.AddToken(t2)
	if err != nil {
		t.Error(err)
	}
	if len(cbtokens) != 2 && cbtokens[0] != t1 {
		t.Error("add token error")
	}
	am2, _ = model.GetAllTokens()
	assert.EqualValues(t, am, am2)

}

func TestGob(t *testing.T) {
	s1 := params.SpectrumTestNetRegistryAddress
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(s1)
	if err != nil {
		t.Error(err)
		return
	}
	encodedData := buf.Bytes()
	fmt.Printf("first\n%s", hex.Dump(encodedData))
	dec := gob.NewDecoder(bytes.NewBuffer(encodedData))
	var sb common.Address
	err = dec.Decode(&sb)
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(s1, sb) {
		t.Error("not equal")
	}
	var buf2 bytes.Buffer
	enc2 := gob.NewEncoder(&buf2)
	enc2.Encode(&sb)
	encodedData2 := buf2.Bytes()
	fmt.Printf("second\n%s", hex.Dump(encodedData2))
	if !reflect.DeepEqual(encodedData, encodedData2) {
		t.Error("not equal")
	}

}
func TestGobAddressMap(t *testing.T) {
	am := make(AddressMap)
	k1 := utils.NewRandomAddress()
	am[k1] = true
	am[utils.NewRandomAddress()] = true
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(am)
	if err != nil {
		t.Error(err)
		return
	}
	encodedData := buf.Bytes()
	dec := gob.NewDecoder(bytes.NewBuffer(encodedData))
	var am2 AddressMap
	err = dec.Decode(&am2)
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(am, am2) {
		t.Error("not equal")
		return
	}
	var buf2 bytes.Buffer
	enc2 := gob.NewEncoder(&buf2)
	enc2.Encode(&am2)
	encodedData2 := buf2.Bytes()
	//should not use bytes equal, because map's random visit mechanism
	//if !reflect.DeepEqual(encodedData, encodedData2) {
	if len(encodedData) != len(encodedData2) {
		t.Errorf("not equal,encodedata=%s,encodedata2=%s", utils.StringInterface(encodedData, 2), utils.StringInterface(encodedData2, 2))
		panic("err")
	}

}
func TestGob2(t *testing.T) {
	for i := 0; i < 50; i++ {
		TestGobAddressMap(t)
	}
}
func TestWithdraw(t *testing.T) {

	model := setupDb(t)
	defer func() {
		model.CloseDB()
	}()
	channel := utils.NewRandomHash()
	secret := utils.ShaSecret(channel[:])
	r := model.IsThisLockHasUnlocked(channel, secret)
	if r == true {
		t.Error("should be false")
		return
	}
	model.UnlockThisLock(channel, secret)
	r = model.IsThisLockHasUnlocked(channel, secret)
	if r == false {
		t.Error("should be true")
		return
	}
	r = model.IsThisLockHasUnlocked(utils.NewRandomHash(), secret)
	if r == true {
		t.Error("shoulde be false")
		return
	}
}

func TestModelDB_IsThisLockRemoved(t *testing.T) {
	model := setupDb(t)
	defer func() {
		model.CloseDB()
	}()
	channel := utils.NewRandomHash()
	secret := utils.ShaSecret(channel[:])
	sender := utils.NewRandomAddress()
	r := model.IsThisLockRemoved(channel, sender, secret)
	if r {
		t.Error("should be false")
		return
	}
	model.RemoveLock(channel, sender, secret)
	r = model.IsThisLockRemoved(channel, sender, secret)
	if !r {
		t.Error("should be true")
		return
	}
}
