package usecase

import (
	"context"

	"github.com/handziurdmytro/messenger/auth-service/internal/domain/auth"
	"github.com/handziurdmytro/messenger/auth-service/internal/port/outbound"
	mock "github.com/stretchr/testify/mock"
)

func NewMockOutboxRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockOutboxRepository {
	if helper, ok := t.(interface{ Helper() }); ok {
		helper.Helper()
	}

	mock := &MockOutboxRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() {
		if helper, ok := t.(interface{ Helper() }); ok {
			helper.Helper()
		}
		mock.AssertExpectations(t)
	})

	return mock
}

type MockOutboxRepository struct {
	mock.Mock
}

type MockOutboxRepository_Expecter struct {
	mock *mock.Mock
}

func (_m *MockOutboxRepository) EXPECT() *MockOutboxRepository_Expecter {
	return &MockOutboxRepository_Expecter{mock: &_m.Mock}
}

func (_mock *MockOutboxRepository) SaveUserCreated(ctx context.Context, user auth.User) error {
	ret := _mock.Called(ctx, user)

	if len(ret) == 0 {
		panic("no return value specified for SaveUserCreated")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, auth.User) error); ok {
		r0 = returnFunc(ctx, user)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

type MockOutboxRepository_SaveUserCreated_Call struct {
	*mock.Call
}

func (_e *MockOutboxRepository_Expecter) SaveUserCreated(ctx any, user any) *MockOutboxRepository_SaveUserCreated_Call {
	return &MockOutboxRepository_SaveUserCreated_Call{Call: _e.mock.On("SaveUserCreated", ctx, user)}
}

func (_c *MockOutboxRepository_SaveUserCreated_Call) Run(run func(ctx context.Context, user auth.User)) *MockOutboxRepository_SaveUserCreated_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 context.Context
		if args[0] != nil {
			arg0 = args[0].(context.Context)
		}
		var arg1 auth.User
		if args[1] != nil {
			arg1 = args[1].(auth.User)
		}
		run(
			arg0,
			arg1,
		)
	})
	return _c
}

func (_c *MockOutboxRepository_SaveUserCreated_Call) Return(err error) *MockOutboxRepository_SaveUserCreated_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *MockOutboxRepository_SaveUserCreated_Call) RunAndReturn(run func(ctx context.Context, user auth.User) error) *MockOutboxRepository_SaveUserCreated_Call {
	_c.Call.Return(run)
	return _c
}

func NewMockTokenRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockTokenRepository {
	if helper, ok := t.(interface{ Helper() }); ok {
		helper.Helper()
	}

	mock := &MockTokenRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() {
		if helper, ok := t.(interface{ Helper() }); ok {
			helper.Helper()
		}
		mock.AssertExpectations(t)
	})

	return mock
}

type MockTokenRepository struct {
	mock.Mock
}

type MockTokenRepository_Expecter struct {
	mock *mock.Mock
}

func (_m *MockTokenRepository) EXPECT() *MockTokenRepository_Expecter {
	return &MockTokenRepository_Expecter{mock: &_m.Mock}
}

func (_mock *MockTokenRepository) DeleteRefresh(ctx context.Context, token auth.RefreshToken) error {
	ret := _mock.Called(ctx, token)

	if len(ret) == 0 {
		panic("no return value specified for DeleteRefresh")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, auth.RefreshToken) error); ok {
		r0 = returnFunc(ctx, token)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

type MockTokenRepository_DeleteRefresh_Call struct {
	*mock.Call
}

func (_e *MockTokenRepository_Expecter) DeleteRefresh(ctx any, token any) *MockTokenRepository_DeleteRefresh_Call {
	return &MockTokenRepository_DeleteRefresh_Call{Call: _e.mock.On("DeleteRefresh", ctx, token)}
}

func (_c *MockTokenRepository_DeleteRefresh_Call) Run(run func(ctx context.Context, token auth.RefreshToken)) *MockTokenRepository_DeleteRefresh_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 context.Context
		if args[0] != nil {
			arg0 = args[0].(context.Context)
		}
		var arg1 auth.RefreshToken
		if args[1] != nil {
			arg1 = args[1].(auth.RefreshToken)
		}
		run(
			arg0,
			arg1,
		)
	})
	return _c
}

func (_c *MockTokenRepository_DeleteRefresh_Call) Return(err error) *MockTokenRepository_DeleteRefresh_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *MockTokenRepository_DeleteRefresh_Call) RunAndReturn(run func(ctx context.Context, token auth.RefreshToken) error) *MockTokenRepository_DeleteRefresh_Call {
	_c.Call.Return(run)
	return _c
}

func (_mock *MockTokenRepository) FindRefresh(ctx context.Context, token auth.RefreshToken) (auth.RefreshEntry, error) {
	ret := _mock.Called(ctx, token)

	if len(ret) == 0 {
		panic("no return value specified for FindRefresh")
	}

	var r0 auth.RefreshEntry
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, auth.RefreshToken) (auth.RefreshEntry, error)); ok {
		return returnFunc(ctx, token)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, auth.RefreshToken) auth.RefreshEntry); ok {
		r0 = returnFunc(ctx, token)
	} else {
		r0 = ret.Get(0).(auth.RefreshEntry)
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, auth.RefreshToken) error); ok {
		r1 = returnFunc(ctx, token)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

type MockTokenRepository_FindRefresh_Call struct {
	*mock.Call
}

func (_e *MockTokenRepository_Expecter) FindRefresh(ctx any, token any) *MockTokenRepository_FindRefresh_Call {
	return &MockTokenRepository_FindRefresh_Call{Call: _e.mock.On("FindRefresh", ctx, token)}
}

func (_c *MockTokenRepository_FindRefresh_Call) Run(run func(ctx context.Context, token auth.RefreshToken)) *MockTokenRepository_FindRefresh_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 context.Context
		if args[0] != nil {
			arg0 = args[0].(context.Context)
		}
		var arg1 auth.RefreshToken
		if args[1] != nil {
			arg1 = args[1].(auth.RefreshToken)
		}
		run(
			arg0,
			arg1,
		)
	})
	return _c
}

func (_c *MockTokenRepository_FindRefresh_Call) Return(refreshEntry auth.RefreshEntry, err error) *MockTokenRepository_FindRefresh_Call {
	_c.Call.Return(refreshEntry, err)
	return _c
}

func (_c *MockTokenRepository_FindRefresh_Call) RunAndReturn(run func(ctx context.Context, token auth.RefreshToken) (auth.RefreshEntry, error)) *MockTokenRepository_FindRefresh_Call {
	_c.Call.Return(run)
	return _c
}

func (_mock *MockTokenRepository) SaveRefresh(ctx context.Context, entry auth.RefreshEntry) error {
	ret := _mock.Called(ctx, entry)

	if len(ret) == 0 {
		panic("no return value specified for SaveRefresh")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, auth.RefreshEntry) error); ok {
		r0 = returnFunc(ctx, entry)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

type MockTokenRepository_SaveRefresh_Call struct {
	*mock.Call
}

func (_e *MockTokenRepository_Expecter) SaveRefresh(ctx any, entry any) *MockTokenRepository_SaveRefresh_Call {
	return &MockTokenRepository_SaveRefresh_Call{Call: _e.mock.On("SaveRefresh", ctx, entry)}
}

func (_c *MockTokenRepository_SaveRefresh_Call) Run(run func(ctx context.Context, entry auth.RefreshEntry)) *MockTokenRepository_SaveRefresh_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 context.Context
		if args[0] != nil {
			arg0 = args[0].(context.Context)
		}
		var arg1 auth.RefreshEntry
		if args[1] != nil {
			arg1 = args[1].(auth.RefreshEntry)
		}
		run(
			arg0,
			arg1,
		)
	})
	return _c
}

func (_c *MockTokenRepository_SaveRefresh_Call) Return(err error) *MockTokenRepository_SaveRefresh_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *MockTokenRepository_SaveRefresh_Call) RunAndReturn(run func(ctx context.Context, entry auth.RefreshEntry) error) *MockTokenRepository_SaveRefresh_Call {
	_c.Call.Return(run)
	return _c
}

func NewMockTransactor(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockTransactor {
	if helper, ok := t.(interface{ Helper() }); ok {
		helper.Helper()
	}

	mock := &MockTransactor{}
	mock.Mock.Test(t)

	t.Cleanup(func() {
		if helper, ok := t.(interface{ Helper() }); ok {
			helper.Helper()
		}
		mock.AssertExpectations(t)
	})

	return mock
}

type MockTransactor struct {
	mock.Mock
}

type MockTransactor_Expecter struct {
	mock *mock.Mock
}

func (_m *MockTransactor) EXPECT() *MockTransactor_Expecter {
	return &MockTransactor_Expecter{mock: &_m.Mock}
}

func (_mock *MockTransactor) WithinTransaction(ctx context.Context, fn func(outbound.Repositories) error) error {
	ret := _mock.Called(ctx, fn)

	if len(ret) == 0 {
		panic("no return value specified for WithinTransaction")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, func(outbound.Repositories) error) error); ok {
		r0 = returnFunc(ctx, fn)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

type MockTransactor_WithinTransaction_Call struct {
	*mock.Call
}

func (_e *MockTransactor_Expecter) WithinTransaction(ctx any, fn any) *MockTransactor_WithinTransaction_Call {
	return &MockTransactor_WithinTransaction_Call{Call: _e.mock.On("WithinTransaction", ctx, fn)}
}

func (_c *MockTransactor_WithinTransaction_Call) Run(run func(ctx context.Context, fn func(outbound.Repositories) error)) *MockTransactor_WithinTransaction_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 context.Context
		if args[0] != nil {
			arg0 = args[0].(context.Context)
		}
		var arg1 func(outbound.Repositories) error
		if args[1] != nil {
			arg1 = args[1].(func(outbound.Repositories) error)
		}
		run(
			arg0,
			arg1,
		)
	})
	return _c
}

func (_c *MockTransactor_WithinTransaction_Call) Return(err error) *MockTransactor_WithinTransaction_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *MockTransactor_WithinTransaction_Call) RunAndReturn(run func(ctx context.Context, fn func(outbound.Repositories) error) error) *MockTransactor_WithinTransaction_Call {
	_c.Call.Return(run)
	return _c
}

func NewMockUserRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockUserRepository {
	if helper, ok := t.(interface{ Helper() }); ok {
		helper.Helper()
	}

	mock := &MockUserRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() {
		if helper, ok := t.(interface{ Helper() }); ok {
			helper.Helper()
		}
		mock.AssertExpectations(t)
	})

	return mock
}

type MockUserRepository struct {
	mock.Mock
}

type MockUserRepository_Expecter struct {
	mock *mock.Mock
}

func (_m *MockUserRepository) EXPECT() *MockUserRepository_Expecter {
	return &MockUserRepository_Expecter{mock: &_m.Mock}
}

func (_mock *MockUserRepository) FindByEmail(ctx context.Context, email string) (auth.User, error) {
	ret := _mock.Called(ctx, email)

	if len(ret) == 0 {
		panic("no return value specified for FindByEmail")
	}

	var r0 auth.User
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string) (auth.User, error)); ok {
		return returnFunc(ctx, email)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, string) auth.User); ok {
		r0 = returnFunc(ctx, email)
	} else {
		r0 = ret.Get(0).(auth.User)
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = returnFunc(ctx, email)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

type MockUserRepository_FindByEmail_Call struct {
	*mock.Call
}

func (_e *MockUserRepository_Expecter) FindByEmail(ctx any, email any) *MockUserRepository_FindByEmail_Call {
	return &MockUserRepository_FindByEmail_Call{Call: _e.mock.On("FindByEmail", ctx, email)}
}

func (_c *MockUserRepository_FindByEmail_Call) Run(run func(ctx context.Context, email string)) *MockUserRepository_FindByEmail_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 context.Context
		if args[0] != nil {
			arg0 = args[0].(context.Context)
		}
		var arg1 string
		if args[1] != nil {
			arg1 = args[1].(string)
		}
		run(
			arg0,
			arg1,
		)
	})
	return _c
}

func (_c *MockUserRepository_FindByEmail_Call) Return(user auth.User, err error) *MockUserRepository_FindByEmail_Call {
	_c.Call.Return(user, err)
	return _c
}

func (_c *MockUserRepository_FindByEmail_Call) RunAndReturn(run func(ctx context.Context, email string) (auth.User, error)) *MockUserRepository_FindByEmail_Call {
	_c.Call.Return(run)
	return _c
}

func (_mock *MockUserRepository) FindByID(ctx context.Context, id auth.UserID) (auth.User, error) {
	ret := _mock.Called(ctx, id)

	if len(ret) == 0 {
		panic("no return value specified for FindByID")
	}

	var r0 auth.User
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, auth.UserID) (auth.User, error)); ok {
		return returnFunc(ctx, id)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, auth.UserID) auth.User); ok {
		r0 = returnFunc(ctx, id)
	} else {
		r0 = ret.Get(0).(auth.User)
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, auth.UserID) error); ok {
		r1 = returnFunc(ctx, id)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

type MockUserRepository_FindByID_Call struct {
	*mock.Call
}

func (_e *MockUserRepository_Expecter) FindByID(ctx any, id any) *MockUserRepository_FindByID_Call {
	return &MockUserRepository_FindByID_Call{Call: _e.mock.On("FindByID", ctx, id)}
}

func (_c *MockUserRepository_FindByID_Call) Run(run func(ctx context.Context, id auth.UserID)) *MockUserRepository_FindByID_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 context.Context
		if args[0] != nil {
			arg0 = args[0].(context.Context)
		}
		var arg1 auth.UserID
		if args[1] != nil {
			arg1 = args[1].(auth.UserID)
		}
		run(
			arg0,
			arg1,
		)
	})
	return _c
}

func (_c *MockUserRepository_FindByID_Call) Return(user auth.User, err error) *MockUserRepository_FindByID_Call {
	_c.Call.Return(user, err)
	return _c
}

func (_c *MockUserRepository_FindByID_Call) RunAndReturn(run func(ctx context.Context, id auth.UserID) (auth.User, error)) *MockUserRepository_FindByID_Call {
	_c.Call.Return(run)
	return _c
}

func (_mock *MockUserRepository) Save(ctx context.Context, user auth.User) error {
	ret := _mock.Called(ctx, user)

	if len(ret) == 0 {
		panic("no return value specified for Save")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, auth.User) error); ok {
		r0 = returnFunc(ctx, user)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

type MockUserRepository_Save_Call struct {
	*mock.Call
}

func (_e *MockUserRepository_Expecter) Save(ctx any, user any) *MockUserRepository_Save_Call {
	return &MockUserRepository_Save_Call{Call: _e.mock.On("Save", ctx, user)}
}

func (_c *MockUserRepository_Save_Call) Run(run func(ctx context.Context, user auth.User)) *MockUserRepository_Save_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 context.Context
		if args[0] != nil {
			arg0 = args[0].(context.Context)
		}
		var arg1 auth.User
		if args[1] != nil {
			arg1 = args[1].(auth.User)
		}
		run(
			arg0,
			arg1,
		)
	})
	return _c
}

func (_c *MockUserRepository_Save_Call) Return(err error) *MockUserRepository_Save_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *MockUserRepository_Save_Call) RunAndReturn(run func(ctx context.Context, user auth.User) error) *MockUserRepository_Save_Call {
	_c.Call.Return(run)
	return _c
}
