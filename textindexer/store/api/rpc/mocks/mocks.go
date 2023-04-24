// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/mycok/uSearch/textindexer/store/api/rpc/proto (interfaces: TextIndexerClient,TextIndexer_SearchClient)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	proto "github.com/mycok/uSearch/textindexer/store/api/rpc/indexproto"
	grpc "google.golang.org/grpc"
	metadata "google.golang.org/grpc/metadata"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// MockTextIndexerClient is a mock of TextIndexerClient interface.
type MockTextIndexerClient struct {
	ctrl     *gomock.Controller
	recorder *MockTextIndexerClientMockRecorder
}

// MockTextIndexerClientMockRecorder is the mock recorder for MockTextIndexerClient.
type MockTextIndexerClientMockRecorder struct {
	mock *MockTextIndexerClient
}

// NewMockTextIndexerClient creates a new mock instance.
func NewMockTextIndexerClient(ctrl *gomock.Controller) *MockTextIndexerClient {
	mock := &MockTextIndexerClient{ctrl: ctrl}
	mock.recorder = &MockTextIndexerClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTextIndexerClient) EXPECT() *MockTextIndexerClientMockRecorder {
	return m.recorder
}

// Index mocks base method.
func (m *MockTextIndexerClient) Index(arg0 context.Context, arg1 *proto.Document, arg2 ...grpc.CallOption) (*proto.Document, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Index", varargs...)
	ret0, _ := ret[0].(*proto.Document)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Index indicates an expected call of Index.
func (mr *MockTextIndexerClientMockRecorder) Index(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Index", reflect.TypeOf((*MockTextIndexerClient)(nil).Index), varargs...)
}

// Search mocks base method.
func (m *MockTextIndexerClient) Search(arg0 context.Context, arg1 *proto.Query, arg2 ...grpc.CallOption) (proto.TextIndexer_SearchClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Search", varargs...)
	ret0, _ := ret[0].(proto.TextIndexer_SearchClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Search indicates an expected call of Search.
func (mr *MockTextIndexerClientMockRecorder) Search(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Search", reflect.TypeOf((*MockTextIndexerClient)(nil).Search), varargs...)
}

// UpdateScore mocks base method.
func (m *MockTextIndexerClient) UpdateScore(arg0 context.Context, arg1 *proto.UpdateScoreRequest, arg2 ...grpc.CallOption) (*emptypb.Empty, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "UpdateScore", varargs...)
	ret0, _ := ret[0].(*emptypb.Empty)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateScore indicates an expected call of UpdateScore.
func (mr *MockTextIndexerClientMockRecorder) UpdateScore(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateScore", reflect.TypeOf((*MockTextIndexerClient)(nil).UpdateScore), varargs...)
}

// MockTextIndexer_SearchClient is a mock of TextIndexer_SearchClient interface.
type MockTextIndexer_SearchClient struct {
	ctrl     *gomock.Controller
	recorder *MockTextIndexer_SearchClientMockRecorder
}

// MockTextIndexer_SearchClientMockRecorder is the mock recorder for MockTextIndexer_SearchClient.
type MockTextIndexer_SearchClientMockRecorder struct {
	mock *MockTextIndexer_SearchClient
}

// NewMockTextIndexer_SearchClient creates a new mock instance.
func NewMockTextIndexer_SearchClient(ctrl *gomock.Controller) *MockTextIndexer_SearchClient {
	mock := &MockTextIndexer_SearchClient{ctrl: ctrl}
	mock.recorder = &MockTextIndexer_SearchClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTextIndexer_SearchClient) EXPECT() *MockTextIndexer_SearchClientMockRecorder {
	return m.recorder
}

// CloseSend mocks base method.
func (m *MockTextIndexer_SearchClient) CloseSend() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloseSend")
	ret0, _ := ret[0].(error)
	return ret0
}

// CloseSend indicates an expected call of CloseSend.
func (mr *MockTextIndexer_SearchClientMockRecorder) CloseSend() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloseSend", reflect.TypeOf((*MockTextIndexer_SearchClient)(nil).CloseSend))
}

// Context mocks base method.
func (m *MockTextIndexer_SearchClient) Context() context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context.
func (mr *MockTextIndexer_SearchClientMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockTextIndexer_SearchClient)(nil).Context))
}

// Header mocks base method.
func (m *MockTextIndexer_SearchClient) Header() (metadata.MD, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Header")
	ret0, _ := ret[0].(metadata.MD)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Header indicates an expected call of Header.
func (mr *MockTextIndexer_SearchClientMockRecorder) Header() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Header", reflect.TypeOf((*MockTextIndexer_SearchClient)(nil).Header))
}

// Recv mocks base method.
func (m *MockTextIndexer_SearchClient) Recv() (*proto.QueryResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Recv")
	ret0, _ := ret[0].(*proto.QueryResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Recv indicates an expected call of Recv.
func (mr *MockTextIndexer_SearchClientMockRecorder) Recv() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Recv", reflect.TypeOf((*MockTextIndexer_SearchClient)(nil).Recv))
}

// RecvMsg mocks base method.
func (m *MockTextIndexer_SearchClient) RecvMsg(arg0 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RecvMsg", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecvMsg indicates an expected call of RecvMsg.
func (mr *MockTextIndexer_SearchClientMockRecorder) RecvMsg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecvMsg", reflect.TypeOf((*MockTextIndexer_SearchClient)(nil).RecvMsg), arg0)
}

// SendMsg mocks base method.
func (m *MockTextIndexer_SearchClient) SendMsg(arg0 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendMsg", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMsg indicates an expected call of SendMsg.
func (mr *MockTextIndexer_SearchClientMockRecorder) SendMsg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMsg", reflect.TypeOf((*MockTextIndexer_SearchClient)(nil).SendMsg), arg0)
}

// Trailer mocks base method.
func (m *MockTextIndexer_SearchClient) Trailer() metadata.MD {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Trailer")
	ret0, _ := ret[0].(metadata.MD)
	return ret0
}

// Trailer indicates an expected call of Trailer.
func (mr *MockTextIndexer_SearchClientMockRecorder) Trailer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Trailer", reflect.TypeOf((*MockTextIndexer_SearchClient)(nil).Trailer))
}
