package ctxvml

import (
	"context"
	"strings"

	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type ctxMarker struct{}

// VmlHeaders contains vml http headers.
type VmlHeaders struct {
	Username string
	OcrCache string
}

func (h VmlHeaders) OcrCacheAllow() bool {
	return strings.ToLower(h.OcrCache) == "allow"
}

var (
	ctxMarkerKey = &ctxMarker{}
)

// UnaryServerInterceptor for propagating client information
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = extractMetadataToContext(ctx)
		return handler(ctx, req)
	}
}

// StreamServerInterceptor for propagating client information
// only on the first request on the stream
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		s := serverStreamWithContext{
			ServerStream: stream,
			ctx:          extractMetadataToContext(ctx),
		}
		return handler(srv, s)
	}
}

type serverStreamWithContext struct {
	grpc.ServerStream
	ctx context.Context
}

func (ss serverStreamWithContext) Context() context.Context {
	return ss.ctx
}

// finds caller information in the gRPC metadata and adds it to the context
func extractMetadataToContext(ctx context.Context) context.Context {
	md, _ := metadata.FromIncomingContext(ctx)
	headers := VmlHeaders{}
	if mdValue, ok := md["vml-username"]; ok && len(mdValue) != 0 {
		headers.Username = mdValue[0]
	}
	if mdValue, ok := md["vml-ocr-cache"]; ok && len(mdValue) != 0 {
		headers.OcrCache = mdValue[0]
	}
	if headers.Username != "" {
		grpc_ctxtags.Extract(ctx).Set("username", headers.Username)
	}
	ctx = context.WithValue(ctx, ctxMarkerKey, headers)
	return ctx
}

// UnaryClientInterceptor propagates any user information from the context
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		outGoingMetadata := packCallerMetadata(ctx)
		for k, v := range outGoingMetadata {
			ctx = metadata.AppendToOutgoingContext(ctx, k, v)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// StreamClientInterceptor propagates any user information from the context
func StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn,
		method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		outGoingMetadata := packCallerMetadata(ctx)
		for k, v := range outGoingMetadata {
			ctx = metadata.AppendToOutgoingContext(ctx, k, v)
		}
		return streamer(ctx, desc, cc, method, opts...)
	}
}

// packCallerMetadata extracts caller specific values from the context,
// into a MD metadata struct that can be propagated with outgoing gRPC requests
func packCallerMetadata(ctx context.Context) map[string]string {
	var md = map[string]string{}
	headers := Extract(ctx)
	md["vml-username"] = headers.Username
	md["vml-ocr-cache"] = headers.OcrCache
	return md
}

// Extract extracts metadate from the context.
func Extract(ctx context.Context) *VmlHeaders {
	headers, ok := ctx.Value(ctxMarkerKey).(VmlHeaders)
	if !ok {
		return &VmlHeaders{}
	}
	return &headers
}

// WithValue Creates context with VML header values
func WithValue(ctx context.Context, headers VmlHeaders) context.Context {
	return context.WithValue(ctx, ctxMarkerKey, headers)
}
