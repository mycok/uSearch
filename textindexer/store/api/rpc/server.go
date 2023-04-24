package rpc

import (
	"context"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mycok/uSearch/textindexer/index"
	proto "github.com/mycok/uSearch/textindexer/store/api/rpc/indexproto"
)

var _ proto.TextIndexerServer = (*TextIndexerServer)(nil)

// TextIndexerServer provides a gRPC wrapper for accessing the index.
type TextIndexerServer struct {
	// Any concrete type that satisfies the index.Indexer interface.
	idx index.Indexer
	proto.UnimplementedTextIndexerServer
}

// NewTextIndexerServer returns a new server instance that uses the provided
// index as its backing store.
func NewTextIndexerServer(idx index.Indexer) *TextIndexerServer {
	return &TextIndexerServer{idx: idx}
}

// Index adds a new document or updates an existing index entry
// in case of an existing document.
func (s *TextIndexerServer) Index(_ context.Context, req *proto.Document) (*proto.Document, error) {
	doc := &index.Document{
		LinkID:  uuidFromBytes(req.LinkId),
		URL:     req.Url,
		Title:   req.Title,
		Content: req.Content,
	}

	err := s.idx.Index(doc)
	if err != nil {
		return nil, err
	}

	req.IndexedAt = timeToProto(doc.IndexedAt)

	return req, nil
}

// Search performs a look up based on query and returns a result
// iterator if successful or an error otherwise.
func (s *TextIndexerServer) Search(req *proto.Query, stream proto.TextIndexer_SearchServer) error {
	query := index.Query{
		Type:       index.QueryType(req.Type),
		Expression: req.Expression,
		Offset:     req.Offset,
	}

	it, err := s.idx.Search(query)
	if err != nil {
		return err
	}

	// Send back the total document count
	countResult := &proto.QueryResult{
		Result: &proto.QueryResult_DocCount{DocCount: it.TotalCount()},
	}

	if err = stream.Send(countResult); err != nil {
		_ = it.Close()
		return err
	}

	// Start streaming
	for it.Next() {
		doc := it.Document()
		res := proto.QueryResult{
			Result: &proto.QueryResult_Doc{
				Doc: &proto.Document{
					LinkId:    doc.LinkID[:],
					Url:       doc.URL,
					Title:     doc.Title,
					Content:   doc.Content,
					IndexedAt: timeToProto(doc.IndexedAt),
				},
			},
		}

		if err = stream.Send(&res); err != nil {
			_ = it.Close()
			return err
		}
	}

	if err = it.Error(); err != nil {
		_ = it.Close()
		return err
	}

	return it.Close()
}

// UpdateScore updates the PageRank score for a document with the
// specified link ID. If no such document exists, a placeholder
// document with the provided score will be created.
func (s *TextIndexerServer) UpdateScore(_ context.Context, req *proto.UpdateScoreRequest) (*emptypb.Empty, error) {
	linkID := uuidFromBytes(req.LinkId)

	return new(emptypb.Empty), s.idx.UpdateScore(linkID, req.PageRankScore)
}

func uuidFromBytes(b []byte) uuid.UUID {
	if len(b) != 16 {
		return uuid.Nil
	}

	var dest uuid.UUID
	_ = copy(dest[:], b)

	return dest
}

func timeToProto(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}
