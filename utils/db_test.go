package utils

import "testing"

func TestPruneMongoDBURIRemovesCredentialsAndPreservesHosts(t *testing.T) {
	uri := "mongodb://user:secret@host1:27017,host2:27018/testdb?replicaSet=rs0&retryWrites=true" // pragma: allowlist secret
	got := PruneMongoDBURI(uri)
	want := "mongodb://host1:27017,host2:27018/testdb?replicaSet=rs0&retryWrites=true"

	if got != want {
		t.Fatalf("PruneMongoDBURI() = %q, want %q", got, want)
	}
}

func TestPruneMongoDBURILeavesCredentialFreeURIUntouched(t *testing.T) {
	uri := "mongodb://host1:27017/testdb?retryWrites=true"
	if got := PruneMongoDBURI(uri); got != uri {
		t.Fatalf("PruneMongoDBURI() = %q, want %q", got, uri)
	}
}
