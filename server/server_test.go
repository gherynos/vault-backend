package server

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	s "vault-backend/store"
)

type ServerTestSuite struct {
	suite.Suite
	pool s.Pool

	creds, auth string
}

func (suite *ServerTestSuite) SetupTest() {

	suite.pool = NewMockPool()
	suite.creds = "testID"
	suite.auth = "Basic " + suite.creds
}

func (suite *ServerTestSuite) TestUnauthorised() {

	req, err := http.NewRequest("GET", "/state/sample", nil)
	if err != nil {

		suite.T().Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := handler{suite.pool, stateHandler}

	handler.ServeHTTP(rr, req)

	assert.Equal(suite.T(), rr.Code, http.StatusUnauthorized)
}

func (suite *ServerTestSuite) TestStateNotFound() {

	req, err := http.NewRequest("GET", "/state/sample", nil)
	if err != nil {

		suite.T().Fatal(err)
	}
	req.Header.Set("Authorization", suite.auth)

	rr := httptest.NewRecorder()
	handler := handler{suite.pool, stateHandler}

	handler.ServeHTTP(rr, req)

	assert.Equal(suite.T(), rr.Code, http.StatusNotFound)
}

func (suite *ServerTestSuite) TestHappyPath() {

	// lock state
	lock := "{\"ID\": \"sampleLocked\"}"
	lReq, lErr := http.NewRequest("LOCK", "/state/sample", strings.NewReader(lock))
	if lErr != nil {

		suite.T().Fatal(lErr)
	}
	lReq.Header.Set("Authorization", suite.auth)

	rr := httptest.NewRecorder()
	handler := handler{suite.pool, stateHandler}

	handler.ServeHTTP(rr, lReq)
	assert.Equal(suite.T(), http.StatusOK, rr.Code)

	store, sErr := suite.pool.Get(suite.creds)

	assert.Nil(suite.T(), sErr)

	data, dErr := store.GetBin("sample-lock")

	assert.Nil(suite.T(), dErr)

	assert.Equal(suite.T(), lock, string(data))

	// store state
	state := "{\"test\": \"value\"}"

	pReq, pErr := http.NewRequest("POST", "/state/sample?ID=sampleLocked", strings.NewReader(state))
	if pErr != nil {

		suite.T().Fatal(pErr)
	}
	pReq.Header.Set("Authorization", suite.auth)

	handler.ServeHTTP(rr, pReq)
	assert.Equal(suite.T(), http.StatusOK, rr.Code)

	st, stErr := store.GetBin("sample")

	assert.Nil(suite.T(), stErr)

	assert.Equal(suite.T(), state, string(st))

	// load state
	gReq, gErr := http.NewRequest("GET", "/state/sample", nil)
	if gErr != nil {

		suite.T().Fatal(gErr)
	}
	gReq.Header.Set("Authorization", suite.auth)

	handler.ServeHTTP(rr, gReq)
	assert.Equal(suite.T(), http.StatusOK, rr.Code)

	assert.Equal(suite.T(), state, rr.Body.String())

	// unlock state
	uReq, uErr := http.NewRequest("UNLOCK", "/state/sample", strings.NewReader(lock))
	if uErr != nil {

		suite.T().Fatal(uErr)
	}
	uReq.Header.Set("Authorization", suite.auth)

	handler.ServeHTTP(rr, uReq)

	assert.Equal(suite.T(), http.StatusOK, rr.Code)

	_, lkErr := store.GetBin("sample-lock")

	assert.NotNil(suite.T(), lkErr)

	assert.Error(suite.T(), lkErr, s.ItemNotFoundError{})
}

func (suite *ServerTestSuite) TestLockedState() {

	// lock state
	lock := "{\"ID\": \"sampleLocked2\"}"
	lReq, lErr := http.NewRequest("LOCK", "/state/sample2", strings.NewReader(lock))
	if lErr != nil {

		suite.T().Fatal(lErr)
	}
	lReq.Header.Set("Authorization", suite.auth)

	rr := httptest.NewRecorder()
	handler := handler{suite.pool, stateHandler}

	handler.ServeHTTP(rr, lReq)
	assert.Equal(suite.T(), http.StatusOK, rr.Code)

	store, sErr := suite.pool.Get(suite.creds)

	assert.Nil(suite.T(), sErr)

	data, dErr := store.GetBin("sample2-lock")

	assert.Nil(suite.T(), dErr)

	assert.Equal(suite.T(), lock, string(data))

	// store state
	state := "{\"test\": \"value2\"}"

	pReq, pErr := http.NewRequest("POST", "/state/sample2?ID=wrongvalue", strings.NewReader(state))
	if pErr != nil {

		suite.T().Fatal(pErr)
	}
	pReq.Header.Set("Authorization", suite.auth)

	handler.ServeHTTP(rr, pReq)
	assert.Equal(suite.T(), http.StatusLocked, rr.Code)

	assert.Equal(suite.T(), lock, strings.Trim(rr.Body.String(), "\n"))

	// lock state again
	lock2 := "{\"ID\": \"sampleLocked3\"}"
	lReq2, lErr2 := http.NewRequest("LOCK", "/state/sample2", strings.NewReader(lock2))
	if lErr2 != nil {

		suite.T().Fatal(lErr2)
	}
	lReq2.Header.Set("Authorization", suite.auth)

	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, lReq2)
	assert.Equal(suite.T(), http.StatusConflict, rr2.Code)

	assert.Equal(suite.T(), lock, strings.Trim(rr2.Body.String(), "\n"))
}

func (suite *ServerTestSuite) TestStoreStateWithoutLocking() {

	// store state
	state := "{\"test\": \"value3\"}"

	pReq, pErr := http.NewRequest("POST", "/state/sample3", strings.NewReader(state))
	if pErr != nil {

		suite.T().Fatal(pErr)
	}
	pReq.Header.Set("Authorization", suite.auth)

	rr := httptest.NewRecorder()
	handler := handler{suite.pool, stateHandler}

	handler.ServeHTTP(rr, pReq)
	assert.Equal(suite.T(), http.StatusUnprocessableEntity, rr.Code)
}

func (suite *ServerTestSuite) TestUnlockingWrongState() {

	// lock state
	lock := "{\"ID\": \"sampleLocked\"}"
	lReq, lErr := http.NewRequest("LOCK", "/state/sample", strings.NewReader(lock))
	if lErr != nil {

		suite.T().Fatal(lErr)
	}
	lReq.Header.Set("Authorization", suite.auth)

	rr := httptest.NewRecorder()
	handler := handler{suite.pool, stateHandler}

	handler.ServeHTTP(rr, lReq)
	assert.Equal(suite.T(), http.StatusOK, rr.Code)

	store, sErr := suite.pool.Get(suite.creds)

	assert.Nil(suite.T(), sErr)

	data, dErr := store.GetBin("sample-lock")

	assert.Nil(suite.T(), dErr)

	assert.Equal(suite.T(), lock, string(data))

	// unlock state
	uReq, uErr := http.NewRequest("UNLOCK", "/state/sample2", strings.NewReader(lock))
	if uErr != nil {

		suite.T().Fatal(uErr)
	}
	uReq.Header.Set("Authorization", suite.auth)

	handler.ServeHTTP(rr, uReq)

	assert.Equal(suite.T(), http.StatusUnprocessableEntity, rr.Code)
}

func TestServerTestSuite(t *testing.T) {

	suite.Run(t, new(ServerTestSuite))
}

// --- Mock Pool ---

type MockPool struct {
	stores map[string]s.Store
}

func NewMockPool() s.Pool {

	p := &MockPool{}
	p.stores = make(map[string]s.Store)

	return p
}

func (p *MockPool) Get(identifier string) (val s.Store, err error) {

	var ok bool
	if val, ok = p.stores[identifier]; ok {

		return

	} else {

		val = NewMockStore()
		p.stores[identifier] = val
		return
	}
}

func (p *MockPool) Delete(identifier string) {

	delete(p.stores, identifier)
}

type MockStore struct {
	data map[string]*[]byte
}

func NewMockStore() s.Store {

	st := &MockStore{}
	st.data = make(map[string]*[]byte)

	return st
}

func (st *MockStore) SetBin(name string, data []byte) error {

	st.data[name] = &data

	return nil
}

func (st *MockStore) GetBin(name string) (out []byte, err error) {

	if val, ok := st.data[name]; ok {

		return *val, nil
	}

	return nil, &s.ItemNotFoundError{}
}

func (st *MockStore) Delete(name string) error {

	if _, ok := st.data[name]; ok {

		delete(st.data, name)
		return nil
	}

	return &s.ItemNotFoundError{}
}
