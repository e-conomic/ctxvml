package ctxvml

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc/metadata"
)

func TestPackMetadata(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "Authorization", "sup3rS3cr37")
	ctx = WithValue(ctx, SsnHeaders{
		Username: "JohnDoe",
	})
	md := packCallerMetadata(ctx)
	if md["x-ssn-username"] != "JohnDoe" {
		t.Fatalf("Unexpected username %s", md["x-ssn-username"])
	}
	for k, v := range md {
		ctx = metadata.AppendToOutgoingContext(ctx, k, v)
	}
	outGoingMD, _ := metadata.FromOutgoingContext(ctx)
	if outGoingMD["authorization"][0] != "sup3rS3cr37" {
		t.Fatalf("Unexpected bearer token %s", outGoingMD["Authorization"])
	}

}
